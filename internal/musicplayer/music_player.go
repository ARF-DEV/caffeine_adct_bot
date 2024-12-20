package musicplayer

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ARF-DEV/caffeine_adct_bot/internal/audio"
	"github.com/ARF-DEV/caffeine_adct_bot/internal/cache"
	"github.com/ARF-DEV/caffeine_adct_bot/utils"
	"github.com/ARF-DEV/caffeine_adct_bot/utils/ytutils"
	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
)

// handle multiple musicplayer for multiple channel and guild (not yet tested)
const (
	audioChannels  = 2
	audioFrameRate = 48000
	audioFrameSize = 960
	audioBitRate   = 64
)

type PlayType uint8

const (
	playShuffle = 1
	playLoop    = 2
	playQueue   = 4
)

type MusicPlayerStream struct {
	queue          []audio.AudioData
	playIdx        int
	nextPlayIdx    int
	switchSoundReq chan int
	mx             *sync.Mutex
	queueAddChan   chan audio.AudioData
	stop           chan struct{}
	play           chan struct{}
	vc             *discordgo.VoiceConnection
	pause          bool
	queueBehaviour PlayType
	ID             string
	r              cache.Cache // to be change
}

func NewMusicPlayer(id string, r cache.Cache) *MusicPlayerStream {
	mps := MusicPlayerStream{
		queue:          []audio.AudioData{},
		playIdx:        0,
		nextPlayIdx:    0,
		mx:             &sync.Mutex{},
		stop:           make(chan struct{}),
		play:           make(chan struct{}),
		queueAddChan:   make(chan audio.AudioData),
		switchSoundReq: make(chan int),
		queueBehaviour: playLoop,
		ID:             id,
		r:              r,
	}

	return &mps
}
func (mps MusicPlayerStream) String() string {
	return fmt.Sprintf("%s, %s, %d, %d, %v", mps.queue, mps.ID, mps.playIdx, mps.nextPlayIdx, mps.vc)
}
func (mps *MusicPlayerStream) Pause() {
	mps.pause = !mps.pause
}

func (mps *MusicPlayerStream) Init() {
	go mps.addToQueueProcess()
	go mps.runplayer()
}

func (mps *MusicPlayerStream) JoinVC(s *discordgo.Session, guildID, channelID string) error {
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}
	mps.vc = vc
	return nil
}

func (mps *MusicPlayerStream) Run() error {
	if len(mps.queue) == 0 {
		return fmt.Errorf("playlist is empty")
	}

	mps.play <- struct{}{}
	return nil

}

func (mps *MusicPlayerStream) SwitchSound(idx int) {
	mps.switchSoundReq <- idx
}
func (mps *MusicPlayerStream) runplayer() {
	<-mps.play

	mps.mx.Lock()
	curMusic := mps.queue[mps.playIdx]
	mps.mx.Unlock()
	mps.nextPlayIdx++

	go curMusic.PlaySoundToVC(mps.vc, &mps.pause)

	defer close(mps.switchSoundReq)
	for {
		select {
		case <-mps.stop:
			return
		case err := <-curMusic.GetFinishChan():
			if err != nil {
				log.Println("error on <-finish: ", err)
				return
			}
			mps.mx.Lock()
			if mps.nextPlayIdx >= len(mps.queue) {
				if mps.queueBehaviour == playLoop {
					mps.nextPlayIdx = 0
				} else {
					log.Println("queue is empty")
				}
			}

			mps.playIdx = mps.nextPlayIdx
			mps.nextPlayIdx++
			curMusic = mps.queue[mps.playIdx]
			mps.mx.Unlock()
			if mps.vc != nil {
				go curMusic.PlaySoundToVC(mps.vc, &mps.pause)
				fmt.Printf("playing: \033[32m%s %d, %d\033[0m\n", curMusic.Title, mps.playIdx, mps.nextPlayIdx)
			}
		case mps.nextPlayIdx = <-mps.switchSoundReq:
			curMusic.Finish(nil)
		default:
			// fmt.Println("cur_music: ", curMusic.Title)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (mps *MusicPlayerStream) addToQueueProcess() {
	defer close(mps.queueAddChan)
	for newSound := range mps.queueAddChan {

		mps.mx.Lock()
		mps.queue = append(mps.queue, newSound)
		mps.mx.Unlock()

		select {
		case <-mps.stop:
			return
		default:
		}
	}
}

func (mps *MusicPlayerStream) GetQueueList() ([]string, int) {
	titles := []string{}
	mps.mx.Lock()
	for _, vid := range mps.queue {
		titles = append(titles, vid.Title)
	}
	mps.mx.Unlock()
	return titles, mps.playIdx
}

func (mps *MusicPlayerStream) AddByURL(url string) {
	ID := utils.GetTYVidIDFromURL(url)
	newAudio := audio.Create("", ID, audio.OpusSound{})

	if err := mps.r.GetAndParse(context.Background(), newAudio.GenRedisKey(), &newAudio); err != nil {
		log.Printf("no-cache detected for %s, downloading...", ID)
		meta, err := ytutils.GetMetaData(url)
		if err != nil {
			panic(err)
		}

		newAudio.Title = meta.Title
		if meta.Type == "playlist" {
			log.Printf("ALERT: user try to download a playlist! %s", url)
			return
		}

		ytDlp := exec.Command("yt-dlp", "-x", "-o", "-", fmt.Sprint(url))
		ytDlpOut, err := ytDlp.StdoutPipe()
		if err != nil {
			panic(err)
		}
		ytDlpErr, err := ytDlp.StderrPipe()
		if err != nil {
			panic(err)
		}

		ffmpeg := exec.Command("ffmpeg", "-v", "debug", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "pipe:1")
		ffmpegIn, err := ffmpeg.StdinPipe()
		if err != nil {
			panic(err)
		}

		ffmpegOut, err := ffmpeg.StdoutPipe()
		if err != nil {
			panic(err)
		}
		defer ffmpegOut.Close()

		if err = ffmpeg.Start(); err != nil {
			panic(err)
		}

		if err = ytDlp.Start(); err != nil {
			panic(err)
		}

		go func() {
			defer ffmpegIn.Close()
			defer ytDlpOut.Close()
			defer ytDlpErr.Close()
			io.Copy(ffmpegIn, ytDlpOut)
		}()

		go func() {
			io.Copy(os.Stdout, ytDlpErr)
		}()

		opusEncoder, err := opus.NewEncoder(audioFrameRate, audioChannels, opus.AppAudio)
		if err != nil {
			panic(err)
		}

		r := bufio.NewReaderSize(ffmpegOut, 16000)
		buf := make([]int16, audioChannels*audioFrameSize)
		i := 0
		for {
			err = binary.Read(r, binary.LittleEndian, buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				if err == io.ErrUnexpectedEOF {
					break
				}
				panic(err)
			}
			opus := make([]byte, 1000)
			n, err := opusEncoder.Encode(buf, opus)
			if err != nil {
				panic(err)
			}
			newAudio.Frames = append(newAudio.Frames, opus[:n])
			i++
		}

		if err = mps.r.SetExp(context.Background(), newAudio.GenRedisKey(), newAudio, 10*time.Minute); err != nil {
			panic(err)
		}
		if err = ytDlp.Wait(); err != nil {
			log.Printf("yt-dlp command finished with error: %v", err)
		}
		if err = ffmpeg.Wait(); err != nil {
			log.Printf("yt-dlp command finished with error: %v", err)
		}
	} else {
		log.Printf("cache for %s detected: adding music from cache", ID)
	}
	mps.queueAddChan <- newAudio
}

func (mps *MusicPlayerStream) PlayAirHorn() {
	mps.Pause()
	audio.AirHornDefault.PlaySound(mps.vc)
	mps.Pause()
}

package musicplayer

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ARF-DEV/caffeine_adct_bot/internal/audio"
	"github.com/ARF-DEV/caffeine_adct_bot/utils/ytutils"
	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
	"github.com/redis/go-redis/v9"
)

// TODO: handle multiple musicplayer for multiple channel and guild (not yet tested)
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
	r              *redis.Client // to be change
}

func NewMusicPlayer(id string, r *redis.Client) *MusicPlayerStream {
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

func (mps *MusicPlayerStream) Run() {
	mps.play <- struct{}{}
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
	meta, err := ytutils.GetMetaData(url)
	if err != nil {
		panic(err)
	}

	if meta.Type == "playlist" {
		log.Printf("ALERT: user try to download a playlist! %s", url)
		return
	}
	newAudio := audio.Create(meta.Title, meta.ID, audio.OpusSound{})

	nFound, _ := mps.r.Exists(context.Background(), newAudio.GenRedisKey()).Result()
	fmt.Println("nFound: ", nFound)

	results, _ := mps.r.Get(context.Background(), newAudio.GenRedisKey()).Bytes()
	// fmt.Println(results)
	if len(results) > 0 {
		fmt.Println("apwkdaowdwk")
		if err := json.Unmarshal([]byte(results), &newAudio.Frames); err != nil {
			panic(err)
		}
	} else {

		fmt.Println("no-cache")
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

		if err = mps.r.Set(context.Background(), newAudio.GenRedisKey(), newAudio.Frames, 10*time.Minute).Err(); err != nil {
			panic(err)
		}
		if err = ytDlp.Wait(); err != nil {
			log.Printf("yt-dlp command finished with error: %v", err)
		}
		if err = ffmpeg.Wait(); err != nil {
			log.Printf("yt-dlp command finished with error: %v", err)
		}
	}

	mps.queueAddChan <- newAudio
}

func (mps *MusicPlayerStream) PlayAirHorn() {
	mps.Pause()
	audio.AirHornDefault.PlaySound(mps.vc)
	mps.Pause()
}

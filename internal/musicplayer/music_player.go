package musicplayer

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/ARF-DEV/caffeine_adct_bot/utils/ytutils"
	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
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
	queue             []AudioData
	playIdx           int
	nextPlayIdx       int
	mx                *sync.Mutex
	queueAddChan      chan AudioData
	stop              <-chan struct{}
	vc                *discordgo.VoiceConnection
	pause             bool
	runInitiated      bool
	addQueueInitiated bool
	queueBehaviour    PlayType
}

func NewMusicPlayer() MusicPlayerStream {
	msp := MusicPlayerStream{
		queue:          []AudioData{},
		playIdx:        0,
		nextPlayIdx:    0,
		mx:             &sync.Mutex{},
		stop:           make(chan struct{}),
		queueAddChan:   make(chan AudioData),
		queueBehaviour: playLoop,
	}

	return msp
}
func (mps *MusicPlayerStream) Pause() {
	mps.pause = !mps.pause
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
	if !mps.runInitiated {
		go mps.run()
		mps.runInitiated = true
	}
	if !mps.addQueueInitiated {
		go mps.addToQueueProcess()
		mps.addQueueInitiated = true
	}
}

func (mps *MusicPlayerStream) InitQueue() {
	if !mps.addQueueInitiated {
		go mps.addToQueueProcess()
		mps.addQueueInitiated = true
	}
}

func (mps *MusicPlayerStream) run() {
	finish := make(chan error, 1)
	finish <- nil
	for {
		select {
		case <-mps.stop:
			return
		case err := <-finish:
			if err != nil {
				log.Println("error on <-finish: ", err)
				return
			}
			mps.mx.Lock()
			mps.playIdx = mps.nextPlayIdx
			if mps.playIdx >= len(mps.queue) {
				if mps.queueBehaviour == playLoop {
					mps.playIdx = 0
				} else {
					log.Println("queue is empty")
				}
				mps.mx.Unlock()
				continue
			}
			curMusic := mps.queue[mps.playIdx]
			mps.mx.Unlock()

			if mps.vc != nil {
				go curMusic.Frames.PlaySoundToVC(finish, mps.vc, &mps.pause)
				mps.nextPlayIdx++

			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (mps *MusicPlayerStream) addToQueueProcess() {
	for {
		select {
		case newSounds := <-mps.queueAddChan:
			mps.mx.Lock()
			mps.queue = append(mps.queue, newSounds)
			mps.mx.Unlock()
		case <-mps.stop:
			return
		default:
			continue
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

// IMPORTANT: IMPLEMENT OAUTH TO AVOID IP BEING BLOCK
func (mps *MusicPlayerStream) AddByURL(url string) {
	// todo handle errors
	meta, err := ytutils.GetMetaData(url)
	if err != nil {
		panic(err)
	}

	// meta := ytutils.YTVideoMeta{}
	if meta.Type == "playlist" {
		log.Printf("ALERT: user try to download a playlist! %s", url)
		return
	}
	newAudio := AudioData{
		ID:    meta.ID,
		Title: meta.Title,
	}
	ytDlp := exec.Command("yt-dlp", "-x", "-o", "-", fmt.Sprint(url))
	ffmpeg := exec.Command("ffmpeg", "-v", "debug", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "pipe:1")
	outp, err := ytDlp.StdoutPipe()
	if err != nil {
		panic(err)
	}

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
		defer outp.Close()
		_, err := io.Copy(ffmpegIn, outp)
		if err != nil {
			log.Fatal("error on io.copy", err)
		}
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
		fmt.Printf("i: %v, err: %v\n", i, err)
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

	mps.queueAddChan <- newAudio
	if err = ytDlp.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}
	if err = ffmpeg.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}

}

func (mps *MusicPlayerStream) PlayAirHorn() {
	mps.Pause()
	airHornDefault.PlaySound(mps.vc)
	mps.Pause()
}

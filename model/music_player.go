package model

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

// TODO: add print lists, and handle multiple musicplayer for multiple channel and guild
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
	playedIdx         int
	mx                *sync.Mutex
	stop              chan bool
	queueAddChan      chan AudioData
	vc                *discordgo.VoiceConnection
	pause             bool
	runInitiated      bool
	addQueueInitiated bool
	queueBehaviour    PlayType
}

func NewMusicPlayer() MusicPlayerStream {
	msp := MusicPlayerStream{
		queue:          []AudioData{},
		playedIdx:      0,
		mx:             &sync.Mutex{},
		stop:           make(chan bool, 1),
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

// func (mps *MusicPlayerStream)
func (mps *MusicPlayerStream) run() {
	for {
		select {
		case <-mps.stop:
			goto breakLoop
		default:
			time.Sleep(100 * time.Millisecond)
		}

		mps.mx.Lock()
		// NOTE: there probably cleaner way to do this
		if len(mps.queue)-mps.playedIdx == 0 {
			if mps.queueBehaviour == playLoop {
				mps.playedIdx = 0
			} else {
				log.Println("queue is empty")
			}
			mps.mx.Unlock()
			continue
		}
		curMusic := mps.queue[mps.playedIdx]
		mps.mx.Unlock()

		if mps.vc != nil {
			// TODO: this still block process, make this concurrent
			err := curMusic.Frames.PlaySoundToVC(mps.vc, &mps.pause)
			if err != nil {
				log.Fatal("error when playing sound")
				break
			}
			mps.playedIdx++
		}
	}
breakLoop:
}

func (mps *MusicPlayerStream) addToQueueProcess() {
	for {
		select {
		case newSounds := <-mps.queueAddChan:
			mps.mx.Lock()
			mps.queue = append(mps.queue, newSounds)
			mps.mx.Unlock()
			fmt.Println("added to queue")
		case <-mps.stop:
			goto breakLoop
		default:
			continue
		}
	}
breakLoop:
}

func (mps *MusicPlayerStream) GetQueueList() ([]string, int) {
	titles := []string{}
	mps.mx.Lock()
	for _, vid := range mps.queue {
		titles = append(titles, vid.Title)
	}
	mps.mx.Unlock()
	return titles, mps.playedIdx
}

func (mps *MusicPlayerStream) AddByURL(url string) {
	meta, err := ytutils.GetMetaData(url)
	if err != nil {
		panic(err)
	}

	if meta.Type == "playlist" {
		log.Printf("ALTER: user try to download a playlist! %s", url)
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
		// vc.OpusSend <- opus[:n]
		newAudio.Frames = append(newAudio.Frames, opus[:n])
		i++
	}

	// mps.mx.Lock()
	// // mps.queue = append(mps.queue, newOpus)
	// mps.mx.Unlock()
	mps.queueAddChan <- newAudio
	log.Println(url, "feed to channel")
	if err = ytDlp.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}
	if err = ffmpeg.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}

}

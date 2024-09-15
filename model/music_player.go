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

	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
)

// TODO: add lists, and handle multiple musicplayer for multiple channel and guild
const (
	audioChannels  = 2
	audioFrameRate = 48000
	audioFrameSize = 960
	audioBitRate   = 64
)

type MusicPlayerStream struct {
	queue        []OpusSound
	playedIdx    int
	mx           *sync.Mutex
	stop         chan bool
	queueAddChan chan OpusSound
	vc           *discordgo.VoiceConnection
}

func NewMusicPlayer() MusicPlayerStream {
	msp := MusicPlayerStream{
		queue:        []OpusSound{},
		playedIdx:    0,
		mx:           &sync.Mutex{},
		stop:         make(chan bool, 1),
		queueAddChan: make(chan OpusSound, 1),
	}

	return msp
}
func (mps *MusicPlayerStream) Run(vc *discordgo.VoiceConnection) {
	mps.vc = vc
	go mps.addToQueueProcess()
	go mps.run(vc)
}
func (mps *MusicPlayerStream) run(vc *discordgo.VoiceConnection) {
	for {
		select {
		case <-mps.stop:
			goto breakLoop
		default:
			time.Sleep(100 * time.Millisecond)
		}

		if len(mps.queue)-mps.playedIdx == 0 {
			log.Println("queue is empty")
			continue
		}
		curMusic := mps.queue[mps.playedIdx]

		err := curMusic.PlaySoundToVC(vc)
		if err != nil {
			log.Fatal("error when playing sound")
			break
		}
		mps.playedIdx++
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
		case <-mps.stop:
			goto breakLoop
		default:
			continue
		}
	}
breakLoop:
}

func (mps *MusicPlayerStream) AddByURL(url string) {
	newOpus := OpusSound{}
	ytDlp := exec.Command("yt-dlp", "-x", "-o", "-", fmt.Sprint(url))

	ffmpeg := exec.Command("ffmpeg", "-v", "debug", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "pipe:1")
	// out := bytes.Buffer{}
	// cmd.Stdout = &out

	//	if err := cmd.Run(); err != nil {
	//		log.Fatal(err)
	//	}
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

	if err = ffmpeg.Start(); err != nil {
		panic(err)
	}

	if err = ytDlp.Start(); err != nil {
		panic(err)
	}

	go func() {
		defer ffmpegIn.Close()
		_, err := io.Copy(ffmpegIn, outp)
		if err != nil {
			log.Fatal("error on io.copy", err)
		}
	}()

	opusEncoder, err := opus.NewEncoder(audioFrameRate, audioChannels, opus.AppAudio)
	if err != nil {
		panic(err)
	}

	// vc, err := s.ChannelVoiceJoin(gID, cID, false, true)
	// if err != nil {
	// 	log.Println("error when connecting to voice channel: ", err)
	// 	return
	// }
	// log.Printf("connecting to c: %s, g: %s SUCCESS \n", cID, gID)

	// defer vc.Disconnect()
	r := bufio.NewReaderSize(ffmpegOut, 16000)
	buf := make([]int16, audioChannels*audioFrameSize)
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
		// vc.OpusSend <- opus[:n]
		newOpus = append(newOpus, opus[:n])

	}

	// mps.mx.Lock()
	// // mps.queue = append(mps.queue, newOpus)
	// mps.mx.Unlock()
	mps.queueAddChan <- newOpus

	if err = ytDlp.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}
	if err = ffmpeg.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}

}

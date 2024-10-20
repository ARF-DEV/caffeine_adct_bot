package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/ARF-DEV/caffeine_adct_bot/config"
	"github.com/ARF-DEV/caffeine_adct_bot/model"
	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
)

const (
	AudioChannels  = 2
	AudioFrameRate = 48000
	AudioFrameSize = 960
	AudioBitRate   = 64

	MaxBytes = (AudioFrameSize * AudioChannels) * 2 // max size of opus data
)

func main() {
	config, err := config.Load("./config.json")
	if err != nil {
		panic(err)
	}

	bot, err := model.NewDisBot(config)
	if err != nil {
		panic(err)
	}
	bot.Open()
	defer bot.Close()

	fmt.Println("running")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	fmt.Println("bot shut down")
}

func streamSound(s *discordgo.Session, link, gID, cID string) {

	ytDlp := exec.Command("yt-dlp", "-x", "-o", "-", fmt.Sprint(link))

	ffmpeg := exec.Command("ffmpeg", "-v", "debug", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "pipe:1")
	// out := bytes.Buffer{}
	// cmd.Stdout = &out

	// if err := cmd.Run(); err != nil {
	// 	log.Fatal(err)
	// }
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

	opusEncoder, err := opus.NewEncoder(AudioFrameRate, AudioChannels, opus.AppAudio)
	if err != nil {
		panic(err)
	}

	vc, err := s.ChannelVoiceJoin(gID, cID, false, true)
	if err != nil {
		log.Println("error when connecting to voice channel: ", err)
		return
	}
	log.Printf("connecting to c: %s, g: %s SUCCESS \n", cID, gID)

	defer vc.Disconnect()
	r := bufio.NewReaderSize(ffmpegOut, 16000)
	buf := make([]int16, AudioChannels*AudioFrameSize)
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
		vc.OpusSend <- opus[:n]

	}

	if err = ytDlp.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}
	if err = ffmpeg.Wait(); err != nil {
		log.Printf("yt-dlp command finished with error: %v", err)
	}

}

// func main() {
// 	// Step 1: Create the yt-dlp command to extract audio as MP3 and output to stdout
// 	ytDlp := exec.Command("yt-dlp", "-x", "--audio-format", "mp3", "-o", "-", "https://www.youtube.com/watch?v=p6HAVF_Yb9M")

// 	// Step 2: Create the ffmpeg command to read from stdin (pipe:0), convert to PCM
// 	ffmpeg := exec.Command("ffmpeg", "-f", "mp3", "-i", "pipe:0", "-f", "s16le", "-ac", "2", "-ar", "48000", "test.pcm")

// 	// Set up pipes for communication between yt-dlp and ffmpeg
// 	ytDlpOut, err := ytDlp.StdoutPipe()
// 	if err != nil {
// 		log.Fatalf("Failed to create yt-dlp stdout pipe: %v", err)
// 	}

// 	ffmpegIn, err := ffmpeg.StdinPipe()
// 	if err != nil {
// 		log.Fatalf("Failed to create ffmpeg stdin pipe: %v", err)
// 	}

// 	// Capture ffmpeg's stderr for debugging
// 	ffmpegStderr, err := ffmpeg.StderrPipe()
// 	if err != nil {
// 		log.Fatalf("Failed to create ffmpeg stderr pipe: %v", err)
// 	}

// 	// Start yt-dlp and ffmpeg processes
// 	if err := ytDlp.Start(); err != nil {
// 		log.Fatalf("Failed to start yt-dlp: %v", err)
// 	}

// 	if err := ffmpeg.Start(); err != nil {
// 		log.Fatalf("Failed to start ffmpeg: %v", err)
// 	}

// 	// Stream data from yt-dlp to ffmpeg
// 	go func() {
// 		_, err := io.Copy(ffmpegIn, ytDlpOut)
// 		if err != nil {
// 			log.Printf("Error piping data from yt-dlp to ffmpeg: %v", err)
// 		}
// 		ffmpegIn.Close() // Important: close stdin to signal ffmpeg that input is complete
// 	}()

// 	// Read and log ffmpeg's stderr for debugging
// 	go func() {
// 		scanner := bufio.NewScanner(ffmpegStderr)
// 		for scanner.Scan() {
// 			log.Printf("FFmpeg: %s", scanner.Text())
// 		}
// 	}()

// 	// Wait for yt-dlp and ffmpeg to finish
// 	if err := ytDlp.Wait(); err != nil {
// 		log.Printf("yt-dlp command finished with error: %v", err)
// 	}

// 	if err := ffmpeg.Wait(); err != nil {
// 		log.Printf("ffmpeg command finished with error: %v", err)
// 	}
// }

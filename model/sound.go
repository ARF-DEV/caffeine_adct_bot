package model

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

type OpusFrame []byte
type OpusSound struct {
	B []OpusFrame
}

func LoadSound(path string) (sound OpusSound, err error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("ooooyy")
		return OpusSound{}, err
	}
	defer file.Close()

	var opusLen int16
	i := 0
	for {

		if err = binary.Read(file, binary.LittleEndian, &opusLen); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return sound, nil
			}
			return sound, fmt.Errorf("error when reading header: %v", err)
		}

		// read opus decoded data
		fmt.Printf("page %v, size : %v\n", i, opusLen)
		opusBuf := make([]byte, opusLen)
		if err = binary.Read(file, binary.LittleEndian, &opusBuf); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				fmt.Println("ooooo")
				return sound, nil
			}
			return sound, fmt.Errorf("error when reading opus data: %v", err)
		}

		sound.B = append(sound.B, opusBuf)
		i++
	}
}

func (opus OpusSound) PlaySound(s *discordgo.Session, guildID, channelID string, packageSize uint64) error {
	fmt.Println("start", guildID, channelID)
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}
	defer vc.Disconnect()
	fmt.Println("connected")
	vc.Speaking(true)
	time.Sleep(200 * time.Millisecond)

	for _, audioData := range opus.B {
		vc.OpusSend <- audioData
	}

	time.Sleep(200 * time.Millisecond)
	vc.Speaking(false)

	return nil
}

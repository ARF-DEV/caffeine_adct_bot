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

var airHornDefault OpusSound

func init() {
	var err error
	airHornDefault, err = LoadSound("airhorn.dca")
	if err != nil {
		panic(err)
	}
}

type OpusFrame []byte
type OpusSound []OpusFrame

func LoadSound(path string) (sound OpusSound, err error) {
	file, err := os.Open(path)
	if err != nil {
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

		opusBuf := make([]byte, opusLen)
		if err = binary.Read(file, binary.LittleEndian, &opusBuf); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return sound, nil
			}
			return sound, fmt.Errorf("error when reading opus data: %v", err)
		}

		sound = append(sound, opusBuf)
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
	vc.Speaking(true)
	time.Sleep(200 * time.Millisecond)

	for _, audioData := range opus {
		vc.OpusSend <- audioData
	}

	time.Sleep(200 * time.Millisecond)
	vc.Speaking(false)

	return nil
}

func (opus OpusSound) PlaySoundToVC(vc *discordgo.VoiceConnection, pause *bool) error {
	vc.Speaking(true)
	time.Sleep(200 * time.Millisecond)

	for _, audioData := range opus {
		for *pause {
			time.Sleep(100 * time.Millisecond)
		}
		vc.OpusSend <- audioData
	}

	time.Sleep(200 * time.Millisecond)
	vc.Speaking(false)

	return nil
}

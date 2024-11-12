package musicplayer

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
type OpusSound []OpusFrame
type AudioData struct {
	Frames OpusSound
	Title  string
	ID     string
}

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

func (opus OpusSound) PlaySound(vc *discordgo.VoiceConnection) error {
	vc.Speaking(true)
	time.Sleep(200 * time.Millisecond)

	for _, audioData := range opus {
		vc.OpusSend <- audioData
	}

	time.Sleep(200 * time.Millisecond)
	vc.Speaking(false)

	return nil
}

func (opus OpusSound) PlaySoundToVC(finish chan<- error, vc *discordgo.VoiceConnection, pause *bool) {
	var err error
	defer func() {
		finish <- err
	}()

	if err = vc.Speaking(true); err != nil {
		return
	}
	time.Sleep(200 * time.Millisecond)

	for _, audioData := range opus {
		for *pause {
			time.Sleep(100 * time.Millisecond)
		}
		vc.OpusSend <- audioData
	}

	time.Sleep(200 * time.Millisecond)
	if err = vc.Speaking(false); err != nil {
		return
	}
}

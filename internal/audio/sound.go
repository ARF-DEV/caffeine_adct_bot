package audio

import (
	"encoding/binary"
	"encoding/json"
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
	finish chan error
}

func (ad AudioData) String() string {
	return fmt.Sprintf("%s %s", ad.Title, ad.ID)
}
func (ad *AudioData) Finish(err error) {
	ad.finish <- err
}

func (ad *AudioData) GetFinishChan() <-chan error {
	return ad.finish
}
func (ad *AudioData) PlaySoundToVC(vc *discordgo.VoiceConnection, pause *bool) {
	var err error

	defer func() {
		ad.finish <- err
	}()

	if err = vc.Speaking(true); err != nil {
		return
	}
	time.Sleep(200 * time.Millisecond)

	for _, audioData := range ad.Frames {
		for *pause {
			time.Sleep(100 * time.Millisecond)
		}
		select {
		case <-ad.finish:
			goto breakLoop
		default:
			vc.OpusSend <- audioData
		}
	}

breakLoop:
	time.Sleep(200 * time.Millisecond)
	if err = vc.Speaking(false); err != nil {
		return
	}

}

func (ad AudioData) GenRedisKey() string {
	return ad.ID
}
func (ad AudioData) MarshalBinary() ([]byte, error) {
	return json.Marshal(ad)
}

func Create(Title, ID string, Frames OpusSound) AudioData {
	return AudioData{
		Title:  Title,
		Frames: Frames,
		ID:     ID,
		finish: make(chan error),
	}
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

func (opus OpusSound) String() string {
	return fmt.Sprintf("opos_sound_len: %d", len(opus))
}

func (opus OpusSound) MarshalBinary() ([]byte, error) {
	return json.Marshal(opus)
}

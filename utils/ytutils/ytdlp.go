package ytutils

import (
	"bufio"
	"encoding/json"
	"io"
	"os/exec"
	"strings"

	"github.com/ARF-DEV/caffeine_adct_bot/utils"
)

type YTVideoMeta struct {
	Title     string   `json:"title"`
	FullTitle string   `json:"fulltitle"`
	ID        string   `json:"id"`
	Type      MetaType `json:"_type"`
}

type MetaType string

const (
	MetaPlayList MetaType = "playlist"
	MetaVideo    MetaType = "video"
)

func GetMetaData(url string) (YTVideoMeta, error) {
	cmdMetaData := exec.Command("yt-dlp", "--skip-download", "--dump-single-json", "--playlist-items", "0", url)

	metaOut, err := cmdMetaData.StdoutPipe()
	if err != nil {
		return YTVideoMeta{}, err
	}

	if err = cmdMetaData.Start(); err != nil {
		return YTVideoMeta{}, err
	}

	r := bufio.NewReader(metaOut)
	metaByte := []byte{}
	for {
		buf := make([]byte, 1000)
		_, err = r.Read(buf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		} else if err != nil {
			return YTVideoMeta{}, err
		}
		metaByte = append(metaByte, buf...)
	}

	trimStr := strings.ReplaceAll(string(metaByte), "\u0000", "")
	meta := YTVideoMeta{}
	if err = json.Unmarshal([]byte(trimStr), &meta); err != nil {
		return YTVideoMeta{}, err
	}
	utils.PrintJSONs("", meta)
	return meta, err
}

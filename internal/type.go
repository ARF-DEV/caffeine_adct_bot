package internal

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

type (
	DisBot struct {
		session *discordgo.Session
		mpMap   map[string]*MusicPlayerStream
		mx      *sync.Mutex
	}
)

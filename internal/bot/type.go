package bot

import (
	"sync"

	"github.com/ARF-DEV/caffeine_adct_bot/internal/cache"
	"github.com/ARF-DEV/caffeine_adct_bot/internal/musicplayer"
	"github.com/bwmarrin/discordgo"
)

type (
	DisBot struct {
		session      *discordgo.Session
		mpMap        map[string]*musicplayer.MusicPlayerStream
		mx           *sync.Mutex
		msgCreateFns map[ActionType]discrodMsgCreateFn
		errors       []string
		r            cache.Cache
	}

	discrodMsgCreateFn func(*discordgo.MessageCreate)
	ActionType         string
)

const (
	WASSAP       ActionType = "wassup"
	AIR_HORN     ActionType = "!airhorn"
	JOIN         ActionType = "join"
	PAUSE        ActionType = "pause"
	PLAY         ActionType = "play"
	ADD          ActionType = "add"
	LIST         ActionType = "list"
	SWITCH_SOUND ActionType = "switch"
)

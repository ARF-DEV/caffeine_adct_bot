package bot

import (
	"sync"

	"github.com/ARF-DEV/caffeine_adct_bot/internal/musicplayer"
	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
)

type (
	DisBot struct {
		session      *discordgo.Session
		mpMap        map[string]*musicplayer.MusicPlayerStream
		mx           *sync.Mutex
		msgCreateFns map[ActionType]discrodMsgCreateFn
		errors       []string
		r            *redis.Client
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

package model

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ARF-DEV/caffeine_adct_bot/config"
	"github.com/bwmarrin/discordgo"
)

type DisBot struct {
	session *discordgo.Session
	mpMap   map[string]*MusicPlayerStream
	mx      *sync.Mutex
}

func NewDisBot(cfg config.Config) (*DisBot, error) {
	b, err := discordgo.New("Bot " + cfg.DiscordAppKey)
	if err != nil {
		return nil, err
	}
	disBot := DisBot{
		session: b,
		mpMap:   map[string]*MusicPlayerStream{},
		mx:      &sync.Mutex{},
	}
	disBot.init()

	return &disBot, nil
}

func (db DisBot) GenerateHandlers() []interface{} {
	return nil
}

func (db *DisBot) GetMusicPlayer(guildID, channelID string) (*MusicPlayerStream, error) {
	key := db.generateMapKey(guildID, channelID)
	mps, found := db.mpMap[key]
	if found {
		return mps, nil
	}

	newMps := NewMusicPlayer()
	if err := newMps.JoinVC(db.session, guildID, channelID); err != nil {
		return nil, err
	}

	db.mx.Lock()
	db.mpMap[key] = &newMps
	db.mx.Unlock()
	return &newMps, nil
}

func (db *DisBot) generateMapKey(guildID, channelID string) string {
	return fmt.Sprintf("%s:%s", guildID, channelID)
}

func (db *DisBot) airHorn(msg *discordgo.MessageCreate) {
	if msg.Content == "!airhorn" {
		db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("okayy <@%s>", msg.Author.ID),
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
			},
		})

		c, err := db.session.State.Channel(msg.ChannelID)
		if err != nil {
			log.Println("Error when finding channel: ", err)
		}
		g, err := db.session.State.Guild(c.GuildID)
		if err != nil {
			log.Println("Error when finding guildID: ", err)
		}

		if err = airHornDefault.PlaySound(db.session, g.ID, c.ID, 320); err != nil {
			log.Printf("Error on playSound(): %v", err)
		}
	}
}

func (db *DisBot) wassup(msg *discordgo.MessageCreate) {
	if msg.Content == "wassup" {
		db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("wassup <@%s>", msg.Author.ID),
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
			},
		})
	}
}

func (db *DisBot) joinVC(msg *discordgo.MessageCreate) {
	if msg.Content == "join" {
		msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
		if err != nil {
			log.Printf("Error on DisBot.GetMusicPlayer(): %v", err)
			return
		}
		if err := msp.JoinVC(db.session, msg.GuildID, msg.ChannelID); err != nil {
			log.Printf("Error on msp.JoinVC(): %v", err)
			return
		}
	}
}

func (db *DisBot) play(msg *discordgo.MessageCreate) {
	if msg.Content == "play" {
		msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
		if err != nil {
			log.Printf("Error on DisBot.GetMusicPlayer(): %v", err)
			return
		}
		msp.Run()
	}
}

func (db *DisBot) pause(msg *discordgo.MessageCreate) {
	if msg.Content == "pause" {
		msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
		if err != nil {
			log.Printf("Error on DisBot.GetMusicPlayer(): %v", err)
			return
		}
		msp.Pause()
	}
}

func (db *DisBot) addMusic(msg *discordgo.MessageCreate) {
	cmds := strings.Split(msg.Content, " ")
	if len(cmds) != 2 {
		db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("<@%s> please use this format 'add <link>'", msg.Author.ID),
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
			},
		})
		return
	}
	msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
	if err != nil {
		log.Printf("Error on DisBot.GetMusicPlayer(): %v", err)
		return
	}
	msp.AddByURL(cmds[1])
	// if we can get the video title that would be awesome
	db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("<@%s> %s has been added!", msg.Author.ID, cmds[1]),
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
		},
	})
	return
}

func (db *DisBot) Open() {
	db.session.Open()
}

func (db *DisBot) Close() {
	db.session.Close()
}

// func (db *DisBot) Run
func (db *DisBot) init() {
	db.session.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMessages | discordgo.IntentGuildVoiceStates
	db.session.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == db.session.State.User.ID {
			return
		}
		strSplit := strings.Split(msg.Content, " ")
		if len(strSplit) == 0 {
			log.Println("empty string")
			return
		}
		handler, found := db.messageCreateDispatch()[strSplit[0]]
		if !found {
			log.Println("handler not implemented")
			return
		}

		handler(msg)
	})
}

func (db *DisBot) messageCreateDispatch() map[string]func(*discordgo.MessageCreate) {
	return map[string]func(*discordgo.MessageCreate){
		"wassup":   db.wassup,
		"!airhorn": db.airHorn,
		"join":     db.joinVC,
		"pause":    db.pause,
		"add":      db.addMusic,
		"play":     db.play,
	}
}

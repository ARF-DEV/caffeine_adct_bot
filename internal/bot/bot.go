package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/ARF-DEV/caffeine_adct_bot/config"
	"github.com/ARF-DEV/caffeine_adct_bot/internal/musicplayer"
	"github.com/bwmarrin/discordgo"
)

func NewDisBot(cfg config.Config) (*DisBot, error) {
	b, err := discordgo.New("Bot " + cfg.DiscordAppKey)
	if err != nil {
		return nil, err
	}
	disBot := DisBot{
		session:      b,
		mpMap:        map[string]*musicplayer.MusicPlayerStream{},
		mx:           &sync.Mutex{},
		msgCreateFns: map[ActionType]discrodMsgCreateFn{},
	}
	disBot.insertMsgCreateFn(WASSAP, disBot.wassup)
	disBot.insertMsgCreateFn(AIR_HORN, disBot.airHorn)
	disBot.insertMsgCreateFn(JOIN, disBot.joinVC)
	disBot.insertMsgCreateFn(PAUSE, disBot.pause)
	disBot.insertMsgCreateFn(ADD, disBot.addMusic)
	disBot.insertMsgCreateFn(PLAY, disBot.play)
	disBot.insertMsgCreateFn(LIST, disBot.printList)
	disBot.insertMsgCreateFn(SWITCH_SOUND, disBot.switchMusic)
	disBot.init()

	return &disBot, nil
}

func (db DisBot) GenerateHandlers() []interface{} {
	return nil
}

// TODO: separate join vc and get pusic player
func (db *DisBot) GetMusicPlayer(guildID, channelID string) (*musicplayer.MusicPlayerStream, error) {
	key := db.generateMapKey(guildID, channelID)
	mps, found := db.mpMap[key]
	if found {
		return mps, nil
	}

	newMps := musicplayer.NewMusicPlayer()
	if err := newMps.JoinVC(db.session, guildID, channelID); err != nil {
		return nil, err
	}
	newMps.InitQueue()

	db.mx.Lock()
	db.mpMap[key] = &newMps
	db.mx.Unlock()

	return &newMps, nil
}

func (db *DisBot) insertMsgCreateFn(actionType ActionType, f discrodMsgCreateFn) {
	db.msgCreateFns[actionType] = f
}
func (db *DisBot) generateMapKey(guildID, channelID string) string {
	return fmt.Sprintf("%s:%s", guildID, channelID)
}

func (db *DisBot) airHorn(msg *discordgo.MessageCreate) {
	db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("okayy <@%s>", msg.Author.ID),
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
		},
	})

	mps, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
	if err != nil {
		db.storeError(err.Error())
		return
	}

	mps.PlayAirHorn()
}

func (db *DisBot) wassup(msg *discordgo.MessageCreate) {
	db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("wassup <@%s>", msg.Author.ID),
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
		},
	})
}

func (db *DisBot) joinVC(msg *discordgo.MessageCreate) {
	msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
	if err != nil {
		return
	}
	if err := msp.JoinVC(db.session, msg.GuildID, msg.ChannelID); err != nil {
		db.storeError(fmt.Sprintf("Error on msp.JoinVC(): %v", err))
		return
	}
}

func (db *DisBot) printList(msg *discordgo.MessageCreate) {
	msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
	if err != nil {
		db.storeError(fmt.Sprintf("Error on DisBot.GetMusicPlayer(): %v", err))
		return
	}
	queueList, playedIdx := msp.GetQueueList()
	messageContent := ""
	for i, title := range queueList {
		messageContent += fmt.Sprintf("%d. %s\n", i+1, title)
	}
	messageContent += fmt.Sprintf("\nCurrently playing: %d. %s", playedIdx+1, queueList[playedIdx])
	db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
		Content: messageContent,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
		},
	})

}

func (db *DisBot) play(msg *discordgo.MessageCreate) {
	msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
	if err != nil {
		db.storeError(fmt.Sprintf("Error on DisBot.GetMusicPlayer(): %v", err))
		return
	}
	msp.Run()
}

func (db *DisBot) switchMusic(msg *discordgo.MessageCreate) {
	cmds := strings.Split(msg.Content, " ")

	if len(cmds) != 2 {
		db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("<@%s> please use this format 'switch <sound-number>' look at command 'list' to see sound-number", msg.Author.ID),
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
			},
		})
		return
	}

	soundNumber, err := strconv.ParseInt(cmds[1], 0, 64)
	if err != nil {
		db.session.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("<@%s> sound-number must be a number", msg.Author.ID),
			AllowedMentions: &discordgo.MessageAllowedMentions{
				Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
			},
		})
		return
	}

	msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
	if err != nil {
		log.Println(err)
		db.storeError(err.Error())
		return
	}

	msp.SwitchSound(int(soundNumber) - 1)

}

func (db *DisBot) pause(msg *discordgo.MessageCreate) {
	msp, err := db.GetMusicPlayer(msg.GuildID, msg.ChannelID)
	if err != nil {
		db.storeError(fmt.Sprintf("Error on DisBot.GetMusicPlayer(): %v", err))
		return
	}
	msp.Pause()
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
		db.storeError(fmt.Sprintf("Error on DisBot.GetMusicPlayer(): %v", err))
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
}

func (db *DisBot) Open() {
	db.session.Open()
}

func (db *DisBot) Close() {
	db.session.Close()
}

func (db *DisBot) init() {
	db.session.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMessages | discordgo.IntentGuildVoiceStates
	db.session.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == db.session.State.User.ID {
			return
		}
		strSplit := strings.Split(msg.Content, " ")
		if len(strSplit) == 0 {
			return
		}
		handler, found := db.msgCreateFns[ActionType(strSplit[0])]
		if !found {
			db.storeError(fmt.Sprintf("handler for action %s are not implemented", strSplit[0]))
			return
		}

		handler(msg)
	})
}

func (db *DisBot) storeError(err string) {
	db.errors = append(db.errors, err)
}

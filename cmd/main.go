package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ARF-DEV/caffeine_adct_bot/config"
	"github.com/ARF-DEV/caffeine_adct_bot/model"
	"github.com/bwmarrin/discordgo"
)

var soundBuf [][]byte = [][]byte{}

func main() {
	config, err := config.Load("./config.json")
	if err != nil {
		panic(err)
	}

	bot, err := discordgo.New("Bot " + config.DiscordAppKey)
	if err != nil {
		panic(err)
	}
	bot.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMessages | discordgo.IntentGuildVoiceStates

	bot.Open()
	defer bot.Close()

	// if err = loadSound(); err != nil {
	// 	panic(err)
	// }
	sound, err := model.LoadSound("airhorn.dca")
	if err != nil {
		panic(err)
	}
	bot.AddHandler(func(s *discordgo.Session, ready *discordgo.Ready) {
		s.UpdateGameStatus(0, "ooyyy")
	})
	bot.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == bot.State.User.ID {
			return
		}

		if msg.Content == "wassup" {
			s.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
				Content: fmt.Sprintf("wassup <@%s>", msg.Author.ID),
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
				},
			})
		}
		if msg.Content == "!airhorn" {
			s.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
				Content: fmt.Sprintf("okayy <@%s>", msg.Author.ID),
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
				},
			})

			c, err := s.State.Channel(msg.ChannelID)
			if err != nil {
				log.Println("Error when finding channel: ", err)
			}
			g, err := s.State.Guild(c.GuildID)
			if err != nil {
				log.Println("Error when finding guildID: ", err)
			}
			// if err = playSound(s, c.ID, g.ID); err != nil {
			// 	log.Printf("Error on playSound(): %v", err)
			// }

			if err = sound.PlaySound(s, g.ID, c.ID, 320); err != nil {
				log.Printf("Error on playSound(): %v", err)
			}
		}
	})

	fmt.Println("running")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("bot shut down")
}

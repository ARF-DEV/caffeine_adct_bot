package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ARF-DEV/caffeine_adct_bot/config"
	"github.com/bwmarrin/discordgo"
)

func main() {
	config, err := config.Load("./config.json")
	if err != nil {
		panic(err)
	}

	discord, err := discordgo.New("Bot " + config.DiscordAppKey)
	if err != nil {
		panic(err)
	}

	discord.Open()
	defer discord.Close()
	discord.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		if msg.Author.ID == discord.State.User.ID {
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
	})

	fmt.Println("running")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("bot shut down")
}

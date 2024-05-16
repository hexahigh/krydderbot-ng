package main

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Command struct {
	Name        string
	Description string
	Usage       string
	Execute     func(*discordgo.Session, *discordgo.MessageCreate, []string)
}

var commands = []Command{}

func initCommands() {
	commands = []Command{
		{
			Name:        "ping",
			Description: "Responds with Pong!",
			Usage:       "ping",
			Execute: func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
				s.ChannelMessageSend(m.ChannelID, "Pong!")
			},
		},
		{
			Name:        "echo",
			Description: "Repeats your message",
			Usage:       "echo <message>",
			Execute: func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
				s.ChannelMessageSend(m.ChannelID, strings.Join(args, " "))
			},
		},
		{
			Name:        "help",
			Description: "Displays help for a command",
			Usage:       "help <command>",
			Execute: func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
				if len(args) == 0 {
					s.ChannelMessageSend(m.ChannelID, "No command given")
					return
				}
				// Find command
				for _, command := range commands {
					if command.Name == args[0] {
						s.ChannelMessageSend(m.ChannelID, getHelp(command))
						return
					}
				}
			},
		},
	}
}

func getHelp(command Command) string {
	help := "Name: " + command.Name +
		"\nDescription: " + command.Description +
		"\nUsage: " + command.Usage
	return help
}

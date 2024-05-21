package main

import (
	"encoding/json"
	"io"
	"net/http"
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
		{
			Name:        "ai",
			Description: "Runs your message through an AI",
			Usage:       "ai <message>",
			Execute: func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
				ai(s, m, strings.Join(args, " "))
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

func ai(s *discordgo.Session, m *discordgo.MessageCreate, msg string) {
	s.ChannelTyping(m.ChannelID)
	type RequestMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type Request struct {
		Messages  []RequestMessage `json:"messages"`
		Mode      string           `json:"mode"`
		Character string           `json:"character"`
	}
	var request Request

	// Get the last 10 messages
	messages, err := s.ChannelMessages(m.ChannelID, 10, "", "", "")
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	for _, message := range messages {
		username := "user"
		if message.Author.Bot {
			username = "assistant"
		}
		request.Messages = append(request.Messages, RequestMessage{Role: username, Content: message.Content})
	}

	request.Mode = "chat"
	request.Character = "krydderbot-ng"
	request.Messages = append(request.Messages, RequestMessage{Role: m.Author.Username, Content: msg})

	requestJson, _ := json.Marshal(request)
	if *params.AiDebug {
		verbosePrintf(3, string(requestJson))
	}

	resp, err := http.Post(*params.AiEndpoint+"/v1/chat/completions", "application/json", strings.NewReader(string(requestJson)))
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	defer resp.Body.Close()

	// Define a struct for the "message" object within each choice
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	// Define a struct for each element in the "choices" array
	type Choice struct {
		Index        int     `json:"index"`
		FinishReason string  `json:"finish_reason"`
		Message      Message `json:"message"`
	}

	// Define a struct for the "usage" object
	type Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	// Define a struct to hold the entire JSON response
	type Response struct {
		ID      string   `json:"id"`
		Object  string   `json:"object"`
		Created int      `json:"created"`
		Model   string   `json:"model"`
		Choices []Choice `json:"choices"`
		Usage   Usage    `json:"usage"`
	}

	var respJson Response

	respBytes, _ := io.ReadAll(resp.Body)

	_ = json.Unmarshal(respBytes, &respJson)
	if *params.AiDebug {
		verbosePrintln(3, string(respBytes))
	}
	s.ChannelMessageSend(m.ChannelID, respJson.Choices[0].Message.Content)

}

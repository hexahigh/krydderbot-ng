package main

import (
	"embed"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	color "github.com/hexahigh/go-lib/ansicolor"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

var (
	b_GitCommit string
	b_BuildTime string
	b_GoVersion string
)

var currentStatusIndex = 0

var (
	triggerWordsContent string
	responsesContent    string
)

//go:embed data/*
var embedData embed.FS

var params Params
var logger *log.Logger

var activities = []discordgo.Activity{
	{
		Name: "Krydder the game",
		Type: discordgo.ActivityTypeGame,
	},
	{
		Name: "the sound of oregano",
		Type: discordgo.ActivityTypeListening,
	},
	{
		Name: "Svømmer i Oregano",
		Type: discordgo.ActivityTypeWatching,
	},
	{
		Name: "You",
		Type: discordgo.ActivityTypeWatching,
	},
}

type Params struct {
	Token      *string
	Verbosity  *int
	Help       *bool
	Version    *bool
	Prefix     *string
	NoColor    *bool
	AiEndpoint *string
	AlwaysAi   *bool
	AiDebug    *bool
}

var verbosityMap = map[int]string{0: "ERROR", 1: "WARN", 2: "INFO", 3: "DEBUG"}

func init() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)

	fs := ff.NewFlagSet("main")

	params.Token = fs.String('t', "token", "", "Bot token")
	params.Verbosity = fs.Int('v', "verbosity", 2, "Verbosity level (0-3)")
	params.Help = fs.Bool('h', "help", "Show this help")
	params.Version = fs.BoolLong("version", "Show version")
	params.Prefix = fs.String('p', "prefix", "^", "Command prefix")
	params.NoColor = fs.BoolLong("no-color", "Don't use colors in log output")
	params.AiEndpoint = fs.StringLong("ai-endpoint", "", "AI Endpoint URL")
	params.AlwaysAi = fs.BoolLong("always-ai", "Always use AI")
	params.AiDebug = fs.BoolLong("ai-debug", "Debug AI")
	_ = fs.String('c', "config", "", "config file (optional)")

	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("KRYDDER"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
	)

	if !*params.NoColor {
		verbosityMap = map[int]string{0: color.Red + "ERROR" + color.Reset, 1: color.Yellow + "WARN" + color.Reset, 2: color.Green + "INFO" + color.Reset, 3: color.Blue + "DEBUG" + color.Reset}
	}

	if *params.Help {
		fmt.Println(ffhelp.Flags(fs), "\nYou can also set options through environment variables, e.g. KRYDDER_TOKEN")
		os.Exit(0)
	}

	if *params.Version {
		fmt.Printf("BUILD_TIME: %s\nGIT_COMMIT: %s\nGO_VERSION: %s\n", b_BuildTime, b_GitCommit, b_GoVersion)
		os.Exit(0)
	}

	if *params.Token == "" {
		verbosePrintln(0, "No token specified")
		os.Exit(1)
	}

	verbosePrintln(2, "Loading trigger words and responses into memory")
	loadTriggerWordsAndResponses()

	verbosePrintln(2, "Initializing commands")
	initCommands()
}

func main() {
	verbosePrintln(2, "Logging in")
	dg, err := discordgo.New("Bot " + *params.Token)
	if err != nil {
		verbosePrintln(0, err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.Ready) {
		verbosePrintln(2, "Bot is ready")
	})

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		verbosePrintln(0, err)
		return
	}

	go cycleStatuses(dg)

	verbosePrintln(2, "Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

}

func changeStatus(s *discordgo.Session, index int) {
	activity := &activities[index]
	s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Activities: []*discordgo.Activity{activity},
	})
}

func cycleStatuses(s *discordgo.Session) {
	currentIndex := currentStatusIndex
	for {
		verbosePrintln(3, "Changing status to", currentIndex)
		changeStatus(s, currentIndex)
		currentIndex = (currentIndex + 1) % len(activities)
		time.Sleep(30 * time.Second)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Return if message is from the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, *params.Prefix) {
		handleCommand(s, m)
		return
	}

	verbosePrintln(3, "Message received in ", m.ChannelID, "with content", m.Content)

	// Only respond if the message is a trigger or the message was in a private dm
	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		verbosePrintln(0, err)
		return
	}
	if isTrigger(m.Message.Content) || channel.Type == discordgo.ChannelTypeDM {

		// If AI is always enabled, then generate the response using ai
		if *params.AlwaysAi {
			ai(s, m, m.Message.Content)
		} else {
			// Otherwise, pick a random response
			_, _ = s.ChannelMessageSend(m.ChannelID, getResponse())
		}

	}
}

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	verbosePrintln(3, "Command received in ", m.ChannelID, "with content", m.Content)

	// Remove the prefix
	prefix := *params.Prefix
	content := m.Content
	if len(content) > len(prefix) {
		content = content[len(prefix):]
	}

	// Split the message
	args := strings.Fields(content)
	name := args[0]
	args = args[1:]

	// Find the command
	for _, command := range commands {
		if command.Name == name {
			// Execute the command
			command.Execute(s, m, args)
			return
		}
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, "Command not found")
}

// TODO: This is a mess, improve code readability
func isTrigger(msg string) bool {
	lowerMsg := strings.ToLower(msg)

	lowerWords := strings.ToLower(triggerWordsContent)

	for _, trigWord := range strings.Fields(lowerWords) {
		// Split the message
		for _, msgWord := range strings.Fields(lowerMsg) {
			if trigWord == msgWord {
				return true
			}
		}
	}
	return false
}

func getResponse() string {
	// Get a random line from the file
	length := len(strings.Split(responsesContent, "\n"))
	return strings.Split(responsesContent, "\n")[rand.Intn(length)]
}

func loadTriggerWordsAndResponses() {
	file, err := embedData.ReadFile("data/triggerwords.txt")
	if err != nil {
		verbosePrintln(0, "Error reading triggerwords.txt: ", err)
		return
	}
	triggerWordsContent = string(file)

	file, err = embedData.ReadFile("data/responses.txt")
	if err != nil {
		verbosePrintln(0, "Error reading responses.txt: ", err)
		return
	}
	responsesContent = string(file)
}

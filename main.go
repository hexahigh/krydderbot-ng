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
	Token     *string
	Verbosity *int
	Help      *bool
	Version   *bool
}

var verbosityMap = map[int]string{0: "ERROR", 1: "WARN", 2: "INFO", 3: "DEBUG"}

func init() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)

	fs := ff.NewFlagSet("main")

	params.Token = fs.String('t', "token", "", "Bot token")
	params.Verbosity = fs.Int('v', "verbosity", 2, "Verbosity level (0-3)")
	params.Help = fs.Bool('h', "help", "Show this help")
	params.Version = fs.BoolLong("version", "Show version")
	_ = fs.StringLong("config", "", "config file (optional)")

	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("KRYDDER"),
	)

	verbosePrintln(3, "Running with these parameters:", params)

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

	verbosePrintln(3, "Message received in ", m.ChannelID, "with content", m.Content)

	if isTrigger(m.Message.Content) {
		_, _ = s.ChannelMessageSend(m.ChannelID, getResponse())
	}
}

func isTrigger(msg string) bool {
	lowerMsg := strings.ToLower(msg)
	// Remove special characters (! , . ?)
	lowerMsg = strings.ReplaceAll(lowerMsg, "!", "")
	lowerMsg = strings.ReplaceAll(lowerMsg, ",", "")
	lowerMsg = strings.ReplaceAll(lowerMsg, ".", "")
	lowerMsg = strings.ReplaceAll(lowerMsg, "?", "")
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

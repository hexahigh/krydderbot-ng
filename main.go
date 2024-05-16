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

	"github.com/bwmarrin/discordgo"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

var (
	b_GitCommit string
	b_Version   string
	b_BuildTime string
	b_GoVersion string
)

//go:embed data/*
var embedData embed.FS

type Params struct {
	Token     *string
	Verbosity *int
	Help      *bool
	Version   *bool
}

var params Params
var logger *log.Logger

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
}

func main() {
	verbosePrintln(2, "Logging in")
	dg, err := discordgo.New("Bot " + *params.Token)
	if err != nil {
		verbosePrintln(0, err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		verbosePrintln(2, "Bot is ready")
	})

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		verbosePrintln(0, err)
		return
	}

	verbosePrintln(2, "Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Return if message is from the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	if isTriggerWord(m.Content) {
		_, _ = s.ChannelMessageSend(m.ChannelID, "krydder")
	}
}

func verbosePrintln(minLevel int, msg ...any) {
	if *params.Verbosity >= minLevel {
		msg = append([]any{"[" + verbosityMap[minLevel] + "]"}, msg...)
		logger.Println(msg...)
	}
}

func verbosePrintf(minLevel int, format string, msg ...any) {
	msg = append([]any{"[" + verbosityMap[minLevel] + "]"}, msg...)
	logger.Printf(format, msg...)
}

func isTriggerWord(msg string) bool {
	// Read triggerwords.txt from embedData
	file, err := embedData.ReadFile("data/triggerwords.txt")
	if err != nil {
		verbosePrintln(0, "Error reading triggerwords.txt: ", err)
		return false
	}
	triggerwords := string(file)

	lowerMsg := strings.ToLower(msg)

	// Iterate over each word in triggerwords
	words := strings.Fields(triggerwords)
	for _, word := range words {
		// Check if the current word exists in the message
		if strings.Contains(lowerMsg, word) {
			return true
		}
	}
	return false
}

func getResponse() string {
	file, err := embedData.ReadFile("data/responses.txt")
	if err != nil {
		verbosePrintln(0, "Error reading responses.txt: ", err)
		return ""
	}

	// Get a random line from the file
	length := len(strings.Split(string(file), "\n"))
	return strings.Split(string(file), "\n")[rand.Intn(length)]
}

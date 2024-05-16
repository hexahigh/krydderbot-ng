package main

import (
	"fmt"
	"strings"
)

func verbosePrintln(minLevel int, msg ...any) {
	if *params.Verbosity >= minLevel {
		msg = sanitize(msg...)
		msg = append([]any{"[" + verbosityMap[minLevel] + "]"}, msg...)
		logger.Println(msg...)
	}
}

func verbosePrintf(minLevel int, format string, msg ...any) {
	msg = sanitize(msg...)
	msg = append([]any{"[" + verbosityMap[minLevel] + "]"}, msg...)
	logger.Printf(format, msg...)
}

func sanitize(msg ...any) []any {
	// Check if the message contains the token
	if *params.Token == "" && len(*params.Token) <= 5 {
		return msg
	}
	if strings.Contains(fmt.Sprint(msg...), *params.Token) {
		// Replace the token with ***
		return []any{strings.Replace(fmt.Sprint(msg...), *params.Token, "***", -1)}
	}
	return msg
}

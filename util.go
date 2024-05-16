package main

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

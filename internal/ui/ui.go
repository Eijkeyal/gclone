package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// NonInteractive makes every question answer itself with "no" (or "skip") without reading
// the terminal. gclone sets it for --ci.
var NonInteractive bool

func AskYesNo(q string, def bool) bool {
	if NonInteractive {
		fmt.Printf("%s (non-interactive, answered no)\n", strings.TrimSpace(q))
		return false
	}
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	fmt.Printf("%s %s ", q, hint)
	line, err := StdinReader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		fmt.Println()
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return def
	case "y", "yes":
		return true
	}
	return false
}

// stdinReader is shared so buffered input is not lost between prompts.
var StdinReader = bufio.NewReader(os.Stdin)

func Choose(max int) int {
	if NonInteractive {
		fmt.Println("Non-interactive: skipped.")
		return 0
	}
	fmt.Printf("Choose [1-%d] (Enter = skip): ", max)
	line, _ := StdinReader.ReadString('\n')
	v, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || v < 1 || v > max {
		return 0
	}
	return v
}

package main

import (
	"log"
	"os/exec"
	"strings"
	"sync"
)

type Notifier struct {
	once             sync.Once
	terminalNotifier string
	warned           bool
}

func (n *Notifier) detect() {
	n.once.Do(func() {
		if p, err := exec.LookPath("terminal-notifier"); err == nil {
			n.terminalNotifier = p
		}
	})
}

// Notify shows a macOS notification. openURL is opened on click when
// terminal-notifier is available; ignored otherwise.
func (n *Notifier) Notify(title, body, openURL string) error {
	n.detect()
	title = sanitize(title)
	body = sanitize(body)

	if n.terminalNotifier != "" {
		args := []string{
			"-title", title,
			"-message", body,
			"-sound", "default",
		}
		if openURL != "" {
			args = append(args, "-open", openURL)
		}
		return exec.Command(n.terminalNotifier, args...).Run()
	}

	if !n.warned {
		log.Printf("notifier: terminal-notifier not found; falling back to osascript (no click-to-open). Install with: brew install terminal-notifier")
		n.warned = true
	}
	script := `display notification ` + quoteAS(body) + ` with title ` + quoteAS(title)
	return exec.Command("osascript", "-e", script).Run()
}

// sanitize strips newlines that would break terminal-notifier display.
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// quoteAS escapes a string for embedding inside an AppleScript double-quoted literal.
func quoteAS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

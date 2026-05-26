package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Notifier struct {
	once   sync.Once
	helper string // path to MotivationNotify executable inside the .app bundle
	warned bool
}

// detect locates the Swift notification helper. Search order:
//  1. $MOTIVATION_HELPER (explicit override)
//  2. ./bin/MotivationNotify.app/Contents/MacOS/MotivationNotify (build output)
//  3. <dir of motivation binary>/MotivationNotify.app/Contents/MacOS/MotivationNotify (installed)
//  4. ~/.local/bin/MotivationNotify.app/Contents/MacOS/MotivationNotify (default install)
func (n *Notifier) detect() {
	n.once.Do(func() {
		if p := os.Getenv("MOTIVATION_HELPER"); p != "" {
			if _, err := os.Stat(p); err == nil {
				n.helper = p
				return
			}
		}
		candidates := []string{
			"bin/MotivationNotify.app/Contents/MacOS/MotivationNotify",
		}
		if exe, err := os.Executable(); err == nil {
			dir := filepath.Dir(exe)
			candidates = append(candidates,
				filepath.Join(dir, "MotivationNotify.app/Contents/MacOS/MotivationNotify"),
				filepath.Join(dir, "bin/MotivationNotify.app/Contents/MacOS/MotivationNotify"),
			)
		}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates,
				filepath.Join(home, ".local/bin/MotivationNotify.app/Contents/MacOS/MotivationNotify"),
			)
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				if abs, err := filepath.Abs(c); err == nil {
					n.helper = abs
				} else {
					n.helper = c
				}
				return
			}
		}
	})
}

// Notify shows a macOS notification. openURL is opened on click when the
// Swift helper is available; ignored by the osascript fallback. emoji, when
// non-empty, is rendered as the notification's icon attachment by the helper.
func (n *Notifier) Notify(title, body, openURL, emoji string) error {
	n.detect()
	title = sanitize(title)
	body = sanitize(body)

	if n.helper != "" {
		args := []string{"-title", title, "-message", body}
		if openURL != "" {
			args = append(args, "-open", openURL)
		}
		if emoji != "" {
			args = append(args, "-emoji", emoji)
		}
		// Start detached so the helper can outlive this call and wait for a click.
		cmd := exec.Command(n.helper, args...)
		return cmd.Start()
	}

	if !n.warned {
		log.Printf("notifier: MotivationNotify helper not found; falling back to osascript (no click-to-open). Build it with: make helper")
		n.warned = true
	}
	script := `display notification ` + quoteAS(body) + ` with title ` + quoteAS(title)
	return exec.Command("osascript", "-e", script).Run()
}

// sanitize strips newlines that would break notification display.
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

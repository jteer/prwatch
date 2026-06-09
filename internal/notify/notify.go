package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// PR fires a macOS system notification for a PR update.
// If terminal-notifier is installed, the notification is clickable and opens url.
// Otherwise falls back to osascript (no click-to-open).
func PR(repo string, number int, title, body, url string) {
	if runtime.GOOS != "darwin" {
		return
	}
	notifTitle := fmt.Sprintf("%s PR: #%d %s", repo, number, title)
	if tryTerminalNotifier(repo, notifTitle, body, url) {
		return
	}
	osascriptNotify(notifTitle, body)
}

func tryTerminalNotifier(appTitle, subtitle, message, url string) bool {
	path, err := exec.LookPath("terminal-notifier")
	if err != nil {
		return false
	}
	args := []string{
		"-title", appTitle,
		"-subtitle", subtitle,
		"-message", message,
		"-sender", "com.apple.Terminal",
	}
	if url != "" {
		args = append(args, "-open", url)
	}
	return exec.Command(path, args...).Run() == nil
}

func osascriptNotify(title, message string) {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`,
		escAS(message), escAS(title))
	_ = exec.Command("osascript", "-e", script).Run()
}

func escAS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

package cli

import "runtime"

// loginFlow is how a sign-in reaches the browser.
type loginFlow int

const (
	flowAuto     loginFlow = iota // chosen from the environment
	flowDevice                    // RFC 8628: a code, any browser
	flowLoopback                  // a browser on this machine
)

// browserElsewhere reports whether a browser opened from here would not reach
// the user, or could not redirect back: an SSH session, or Linux or BSD
// without a display.
func (a *App) browserElsewhere() bool {
	if a.Getenv("SSH_CONNECTION") != "" || a.Getenv("SSH_TTY") != "" {
		return true
	}
	goos := a.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goos {
	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly":
		return a.Getenv("DISPLAY") == "" && a.Getenv("WAYLAND_DISPLAY") == ""
	}
	return false
}

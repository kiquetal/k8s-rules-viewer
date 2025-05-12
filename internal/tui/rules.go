package tui

import (
	"os"
	"strings"
)

// StatusSymbols provides both emoji and text fallbacks for statuses
type StatusSymbols struct {
	Success string
	Failure string
}

// GetStatusSymbols returns appropriate status symbols based on terminal capabilities
func GetStatusSymbols() StatusSymbols {
	// Check if terminal likely supports emoji
	// TERM_PROGRAM environment variable is set by many terminal emulators
	termProgram := os.Getenv("TERM_PROGRAM")
	colorTerm := os.Getenv("COLORTERM")
	term := os.Getenv("TERM")

	// These terminals generally have good Unicode/emoji support
	goodTerms := []string{"iTerm.app", "Apple_Terminal", "vscode", "hyper", "alacritty", "kitty", "terminator"}

	useEmoji := false

	// Check if we're in a terminal with likely emoji support
	if termProgram != "" {
		for _, t := range goodTerms {
			if strings.Contains(strings.ToLower(termProgram), strings.ToLower(t)) {
				useEmoji = true
				break
			}
		}
	}

	// Additional checks for terminal types that might support emoji
	if !useEmoji && (strings.Contains(colorTerm, "truecolor") ||
		strings.Contains(colorTerm, "24bit") ||
		strings.Contains(term, "xterm-256color")) {
		useEmoji = true
	}

	if useEmoji {
		return StatusSymbols{
			Success: "✅",
			Failure: "❌",
		}
	}

	// Fallback to ASCII symbols
	return StatusSymbols{
		Success: "[+]",
		Failure: "[!]",
	}
}

// GetRulesCompliance returns information about Kubernetes rules compliance
func GetRulesCompliance() string {
	symbols := GetStatusSymbols()

	return strings.Join([]string{
		symbols.Success + " Resource Limits: Set correctly",
		symbols.Success + " Liveness Probe: Configured",
		symbols.Success + " Readiness Probe: Configured",
		symbols.Failure + " Security Context: Not set",
		symbols.Success + " Network Policies: Applied",
		symbols.Success + " RBAC: Properly configured",
	}, "\n")
}

package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Color functions - these respect NoColor setting automatically
var (
	cyan      = color.New(color.FgCyan)
	green     = color.New(color.FgGreen)
	boldGreen = color.New(color.FgGreen, color.Bold)
	magenta   = color.New(color.FgMagenta)
	red       = color.New(color.FgRed)
	yellow    = color.New(color.FgYellow)
	dim       = color.New(color.Faint)
)

// Branch returns a branch name in cyan
func Branch(name string) string {
	return cyan.Sprint(name)
}

// CurrentBranchMarker returns the bold green asterisk for current branch
func CurrentBranchMarker() string {
	return boldGreen.Sprint(" *")
}

// PRState returns the PR state with appropriate coloring
func PRState(state string) string {
	switch strings.ToUpper(state) {
	case "OPEN":
		return green.Sprint(strings.ToLower(state))
	case "MERGED":
		return magenta.Sprint(strings.ToLower(state))
	case "CLOSED":
		return red.Sprint(strings.ToLower(state))
	default:
		return strings.ToLower(state)
	}
}

// Success returns a green success message with checkmark
func Success(msg string) string {
	return green.Sprintf("✓ %s", msg)
}

// Warning returns a yellow warning message with warning sign
func Warning(msg string) string {
	return yellow.Sprintf("⚠ %s", msg)
}

// Error returns a red error message with X
func Error(msg string) string {
	return red.Sprintf("✗ %s", msg)
}

// ErrorText returns red text without the X prefix
func ErrorText(msg string) string {
	return red.Sprint(msg)
}

// Command returns a command in green (for help text)
func Command(cmd string) string {
	return green.Sprint(cmd)
}

// Dim returns dimmed/gray text
func Dim(s string) string {
	return dim.Sprint(s)
}

// Progress returns progress indicators like (1/5) in dim
func Progress(current, total int) string {
	return dim.Sprintf("(%d/%d)", current, total)
}

// Pipe returns the tree pipe character in dim
func Pipe() string {
	return dim.Sprint("|")
}

// TreeNodeCurrent returns the filled circle for current branch (bold green)
func TreeNodeCurrent() string {
	return boldGreen.Sprint("●")
}

// TreeNode returns the hollow circle for non-current branches (dim)
func TreeNode() string {
	return dim.Sprint("○")
}

// TreeLine returns the vertical line for connecting branches (dim)
func TreeLine() string {
	return dim.Sprint("│")
}

// SuccessIcon returns just the green checkmark
func SuccessIcon() string {
	return green.Sprint("✓")
}

// WarningIcon returns just the yellow warning sign
func WarningIcon() string {
	return yellow.Sprint("⚠")
}

// ErrorIcon returns just the red X
func ErrorIcon() string {
	return red.Sprint("✗")
}

// PRInfo formats PR information with URL and colored state
func PRInfo(url, state string) string {
	return fmt.Sprintf("[%s :%s]", url, PRState(state))
}

// SetNoColor sets whether color output is disabled
func SetNoColor(disabled bool) {
	color.NoColor = disabled
}

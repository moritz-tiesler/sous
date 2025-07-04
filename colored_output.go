package main

import (
	"fmt"

	"github.com/fatih/color"
)

// PrintThink prints a message in the "Think" color theme (bright yellow, bold)
func PrintThink(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	color.New(color.FgHiYellow).Print(msg)
}

// PrintNonThink prints a message in the "Non-Think" color theme (cyan, bold)
func PrintNonThink(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	color.New(color.FgCyan, color.Bold).Print(msg)
}

// PrintAction prints a message in the "Action" color theme (magenta, italic)
func PrintAction(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	color.New(color.FgHiMagenta, color.Italic, color.Bold).Print(msg)
}

package tui

import (
	"fmt"
)

// StatusPrinter prints formatted status messages to the terminal.
type StatusPrinter struct{}

// NewStatusPrinter creates a new StatusPrinter.
func NewStatusPrinter() *StatusPrinter {
	return &StatusPrinter{}
}

// Success prints a success message with a checkmark.
func (p *StatusPrinter) Success(msg string) {
	fmt.Printf("  %s %s\n", CheckMark, msg)
}

// Warning prints a warning message.
func (p *StatusPrinter) Warning(msg string) {
	fmt.Printf("  %s %s\n", WarnMark, WarningStyle.Render(msg))
}

// Error prints an error message.
func (p *StatusPrinter) Error(msg string) {
	fmt.Printf("  %s %s\n", CrossMark, ErrorStyle.Render(msg))
}

// Info prints an info message with an arrow.
func (p *StatusPrinter) Info(msg string) {
	fmt.Printf("  %s %s\n", ArrowMark, msg)
}

// Step prints a step in a process.
func (p *StatusPrinter) Step(step, total int, msg string) {
	counter := MutedStyle.Render(fmt.Sprintf("[%d/%d]", step, total))
	fmt.Printf("  %s %s\n", counter, msg)
}

// Section prints a section header.
func (p *StatusPrinter) Section(title string) {
	fmt.Println()
	fmt.Println(BoldStyle.Render("  " + title))
}

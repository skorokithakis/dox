package utils

import (
	"os"

	"golang.org/x/term"
)

// GetTerminalSize returns the current terminal width and height.
// Returns 0, 0 if the terminal size cannot be determined.
func GetTerminalSize() (width, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Default to standard terminal size if we can't get the actual size
		return 80, 24
	}
	return width, height
}

// SetupTerminal configures the terminal for raw mode if TTY is detected.
// Returns the original terminal state to be restored later.
func SetupTerminal() (*term.State, error) {
	// Check if stdin is a terminal.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, nil
	}

	// Save the current state so we can restore it later.
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	// Set raw mode for immediate keystroke passing.
	_, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	return oldState, nil
}

// RestoreTerminal restores the terminal to its original state.
func RestoreTerminal(oldState *term.State) {
	if oldState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}
}
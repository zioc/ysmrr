// Package tput provides convenience functions for sending escape sequences to the terminal.
// The escape codes used have been derrived from the tput program.
package tput

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Sc saves the current position of the cursor.
func Sc(w io.Writer) {
	Write(w, "\u001b7")
}

// Rc restores the cursor to the saved position.
func Rc(w io.Writer) {
	Write(w, "\u001b8")
}

// Civis hides the cursor.
func Civis(w io.Writer) {
	Write(w, "\u001b[?25l")
}

// Cnorm shows the cursor.
func Cnorm(w io.Writer) {
	Writef(w, "\u001b[?25h")
}

// Cuu moves the cursor up by n lines.
func Cuu(w io.Writer, n int) {
	Writef(w, "\u001b[%dA", n)
}

// BufScreen ensures that there are enough lines available
// by sending n * newlines to the writer.
func BufScreen(w io.Writer, n int) {
	Writef(w, "%s", strings.Repeat("\n", n))
}

func ClearLine(w io.Writer) {
	Write(w, "\u001b[K")
}

func TtySize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0
	} else {
		return width, height
	}
}

func Tty() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) || os.Getenv("YSMRR_FORCE_TTY") == "true"
}

func Write(w io.Writer, s string) {
	if Tty() {
		fmt.Fprint(w, s)
	}
}

func Writef(w io.Writer, format string, a ...interface{}) {
	if Tty() {
		fmt.Fprintf(w, format, a...)
	}
}

// Package ysmrr provides a simple interface for creating and managing
// multiple spinners.
package ysmrr

import (
	"context"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/zioc/ysmrr/pkg/animations"
	"github.com/zioc/ysmrr/pkg/colors"
	"github.com/zioc/ysmrr/pkg/tput"
)

// SpinnerManager manages spinners
type SpinnerManager interface {
	AddSpinner(msg string) *Spinner
	GetSpinners() []*Spinner
	SetSpinnersCount(count int)
	GetWriter() io.Writer
	GetAnimation() []string
	GetFrameDuration() time.Duration
	GetSpinnerColor() colors.Color
	GetErrorColor() colors.Color
	GetCompleteColor() colors.Color
	GetMessageColor() colors.Color
	Start()
	Stop()
}

type spinnerManager struct {
	spinners []*Spinner
	mutex    sync.RWMutex

	chars         []string
	frameDuration time.Duration
	spinnerColor  colors.Color
	completeColor colors.Color
	errorColor    colors.Color
	messageColor  colors.Color
	writer        io.Writer
	context       context.Context
	done          chan bool
	hasUpdate     chan bool
	ticks         *time.Ticker
	lines         int
	frame         int
	tty           bool
}

// AddSpinner adds a new spinner to the manager.
func (sm *spinnerManager) AddSpinner(message string) *Spinner {
	spinner := sm.createSpinner(message)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.spinners = append(sm.spinners, spinner)

	return spinner
}

// Create a new spinner with manager options.
func (sm *spinnerManager) createSpinner(message string) *Spinner {
	opts := SpinnerOptions{
		Message:       message,
		SpinnerColor:  sm.spinnerColor,
		CompleteColor: sm.completeColor,
		ErrorColor:    sm.errorColor,
		MessageColor:  sm.messageColor,
		HasUpdate:     sm.hasUpdate,
	}
	return NewSpinner(opts)
}

// GetSpinners returns the spinners managed by the manager.
func (sm *spinnerManager) GetSpinners() []*Spinner {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.spinners
}

// SetSpinnersCount defines the amount of spinners managed by the manager.
// It will create empty sprinners or delete last ones to match requested size.
func (sm *spinnerManager) SetSpinnersCount(count int) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	if count > len(sm.spinners) {
		for i:=len(sm.spinners); i < count; i++{
			sm.spinners = append(sm.spinners, sm.createSpinner(""))
		}
	} else{
		sm.spinners = sm.spinners[:count]
	}
}

// Start the rendering loop in a goroutine.
// Creates a interrupt signal if cancel context was not provided.
func (sm *spinnerManager) Start() {
	if sm.context == nil {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		go func() {
			<-ctx.Done()
			sm.Stop()
			cancel()
			os.Exit(0)
		}()
	}
	sm.ticks = time.NewTicker(sm.frameDuration)
	go sm.render()
}

// Stop sends a signal to the render goroutine to stop
// rendering. We then stop the ticker and persist the final
// frame for each spinner.
// Finally the deferred tput command will ensure tat the cursor
// is no longer hidden.
func (sm *spinnerManager) Stop() {
	sm.done <- true
	sm.ticks.Stop()

	// Persist the final frame for each spinner.
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	for _, s := range sm.spinners {
		tput.ClearLine(sm.writer)
		s.Print(sm.writer, sm.chars[sm.frame])
	}
	tput.Cnorm(sm.writer)
}

// GetWriter returns the configured io.Writer.
func (sm *spinnerManager) GetWriter() io.Writer {
	return sm.writer
}

// GetAnimation returns the current spinner animation as
// a slice of strings.
func (sm *spinnerManager) GetAnimation() []string {
	return sm.chars
}

// GetFrameDuration returns the configured frame duration.
func (sm *spinnerManager) GetFrameDuration() time.Duration {
	return sm.frameDuration
}

// GetSpinnerColor returns the configured color of the spinners.
func (sm *spinnerManager) GetSpinnerColor() colors.Color {
	return sm.spinnerColor
}

// GetErrorColor returns the configured color of error icon.
func (sm *spinnerManager) GetErrorColor() colors.Color {
	return sm.errorColor
}

// GetCompleteColor returns the configured color of completed icon.
func (sm *spinnerManager) GetCompleteColor() colors.Color {
	return sm.completeColor
}

// GetMessageColor returns the color of the message.
func (sm *spinnerManager) GetMessageColor() colors.Color {
	return sm.messageColor
}

// This is the code that actually renders the spinners.
// Rendering is done in a separate goroutine so that the main
// goroutine can continue to handle signals.
// The render goroutine is called by Start().
//
// Each tick signal calls renderFrame which in turn will print the current
// frame to the writer provided by the manager.
//
// The render method also emits tput strings to the terminal to set the
// correct location of the cursor.
func (sm *spinnerManager) render() {
	// Prepare the screen.
	tput.Civis(sm.writer)
	defer tput.Cnorm(sm.writer)

	for {
		select {
		case <-sm.done:
			return
		case <-sm.hasUpdate:
			sm.renderFrame(false)
		case <-sm.ticks.C:
			sm.renderFrame(true)
		}

		tput.Rc(sm.writer)
	}
}

func (sm *spinnerManager) renderFrame(animate bool) {
	if !sm.tty {
		return
	}

	spinners := sm.GetSpinners()
	linesCount := len(spinners)
	width, height := tput.TtySize()
	if width != 0 {
		linesCount = 0
		for _, s := range spinners {
			linesCount = linesCount + (len([]rune(s.message))+3)/width
			if (len([]rune(s.message))+3)%width != 0 {
				linesCount = linesCount + 1
			}
		}
	}
	if linesCount > sm.lines {
		sm.lines = linesCount
	} else if height != 0 && sm.lines > height {
		sm.lines = height
	}

	tput.BufScreen(sm.writer, linesCount)
	tput.Cuu(sm.writer, linesCount)
	tput.Sc(sm.writer)

	for _, s := range spinners {
		tput.ClearLine(sm.writer)
		s.Print(sm.writer, sm.chars[sm.frame])
	}
	// Clear extra lines if any
	for i := linesCount; i < sm.lines; i++ {
		tput.ClearLine(sm.writer)
		if i+1 < sm.lines {
			tput.Write(sm.writer, "\n")
		}
	}

	if animate {
		sm.setNextFrame()
	}
}

func (sm *spinnerManager) setNextFrame() {
	sm.frame += 1
	if sm.frame >= len(sm.chars) {
		sm.frame = 0
	}
}

// NewSpinnerManager is the constructor for the SpinnerManager.
// You can create a new manager with sensible defaults or you can
// pass in your own options using the provided methods.
//
// For example, this will initialize a default manager:
//
//	sm := NewSpinnerManager()
//
// Or this will initialize a manager with a custom animation:
//
//	sm := NewSpinnerManager(
//		WithAnimation(animations.Arrows)
//	)
//
// You can pass in multiple options to the constructor:
//
//	sm := NewSpinnerManager(
//		WithAnimation(animations.Arrows),
//		WithFrameDuration(time.Millisecond * 100),
//		WithSpinnerColor(colors.Red),
//	)
func NewSpinnerManager(options ...managerOption) SpinnerManager {
	animationSpeed, animationChars := animations.GetAnimation(animations.Dots)
	sm := &spinnerManager{
		chars:         animationChars,
		frameDuration: animationSpeed,
		spinnerColor:  colors.FgHiGreen,
		errorColor:    colors.FgHiRed,
		completeColor: colors.FgHiGreen,
		messageColor:  colors.NoColor,
		writer:        getWriter(),
		done:          make(chan bool),
		hasUpdate:     make(chan bool),
		tty:           tput.Tty(),
		lines:         0,
	}

	for _, option := range options {
		option(sm)
	}
	return sm
}

func getWriter() io.Writer {
	// Windows support conveniently provided by github.com/mattn/go-colorable <3.
	if runtime.GOOS == "windows" {
		return colorable.NewColorableStdout()
	} else {
		return os.Stdout
	}
}

// Option represents a spinner manager option.
type managerOption func(*spinnerManager)

// WithContext sets the context for spinner
func WithContext(context context.Context) managerOption {
	return func(sm *spinnerManager) {
		sm.context = context
	}
}

// WithAnimation sets the animation used for the spinners.
// Available spinner types can be found in the package github.com/zioc/ysmrr/pkg/animations.
// The default spinner animation is the Dots.
func WithAnimation(a animations.Animation) managerOption {
	return func(sm *spinnerManager) {
		animationSpeed, animationChars := animations.GetAnimation(a)
		sm.chars = animationChars
		sm.frameDuration = animationSpeed
	}
}

// WithFrameDuration sets the duration of each frame.
// The default duration is 250 milliseconds.
func WithFrameDuration(d time.Duration) managerOption {
	return func(sm *spinnerManager) {
		sm.frameDuration = d
	}
}

// WithSpinnerColor sets the color of the spinners.
// Available colors can be found in the package github.com/zioc/ysmrr/pkg/colors.
// The default color is FgHiGreen.
func WithSpinnerColor(c colors.Color) managerOption {
	return func(sm *spinnerManager) {
		sm.spinnerColor = c
	}
}

// WithErrorColor sets the color of the error icon.
// Available colors can be found in the package github.com/zioc/ysmrr/pkg/colors.
// The default color is FgHiRed.
func WithErrorColor(c colors.Color) managerOption {
	return func(sm *spinnerManager) {
		sm.errorColor = c
	}
}

// WithCompleteColor sets the color of the complete icon.
// Available colors can be found in the package github.com/zioc/ysmrr/pkg/colors.
// The default color is FgHiGreen.
func WithCompleteColor(c colors.Color) managerOption {
	return func(sm *spinnerManager) {
		sm.completeColor = c
	}
}

// WithMessageColor sets the color of the message.
// Available colors can be found in the package github.com/zioc/ysmrr/pkg/colors.
// The default color is NoColor.
func WithMessageColor(c colors.Color) managerOption {
	return func(sm *spinnerManager) {
		sm.messageColor = c
	}
}

// WithWriter sets the writer used for the spinners.
// The writer can be anything that implements the io.Writer interface.
// The default writer is os.Stdout.
func WithWriter(w io.Writer) managerOption {
	return func(sm *spinnerManager) {
		sm.writer = w
	}
}

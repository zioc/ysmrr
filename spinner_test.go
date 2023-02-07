package ysmrr_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/zioc/ysmrr"
	"github.com/zioc/ysmrr/pkg/animations"
	"github.com/zioc/ysmrr/pkg/colors"
	"github.com/stretchr/testify/assert"
)

var initialMessage = "test"
var initialOpts = ysmrr.SpinnerOptions{
	Message:       initialMessage,
	SpinnerColor:  colors.FgHiGreen,
	CompleteColor: colors.FgHiGreen,
	ErrorColor:    colors.FgHiRed,
	MessageColor:  colors.NoColor,
	HasUpdate:     make(chan bool),
}

func TestNewSpinner(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	assert.NotNil(t, spinner)
}

func TestSpinnerGetMessage(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	assert.Equal(t, initialMessage, spinner.GetMessage())
}

func TestSpinnerIsError(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	assert.Equal(t, false, spinner.IsError())
}

func TestSpinnerIsComplete(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	assert.Equal(t, false, spinner.IsComplete())
}

func TestSpinnerUpdateMessage(t *testing.T) {
	updatedMessage := "updated message"
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	spinner.UpdateMessage(updatedMessage)
	assert.Equal(t, updatedMessage, spinner.GetMessage())
}

func TestSpinnerUpdateMessagef(t *testing.T) {
	expectedMessage := "updated message test"
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	spinner.UpdateMessagef("updated message %s", "test")
	assert.Equal(t, expectedMessage, spinner.GetMessage())
}

func TestSpinnerComplete(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	spinner.Complete()
	assert.Equal(t, true, spinner.IsComplete())
}

func TestSpinnerError(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	spinner.Error()
	assert.Equal(t, true, spinner.IsError())
}

func TestPrint(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)

	var buf bytes.Buffer
	_, dots := animations.GetAnimation(animations.Dots)
	spinner.Print(&buf, dots[0])

	want := fmt.Sprintf("%s %s\r\n", dots[0], initialMessage)
	assert.Equal(t, want, buf.String())
}

func TestPrintWithComplete(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	spinner.Complete()

	var buf bytes.Buffer
	spinner.Print(&buf, "✓")

	want := fmt.Sprintf("%s %s\r\n", "✓", initialMessage)
	assert.Equal(t, want, buf.String())
}

func TestPrintWithError(t *testing.T) {
	opts := initialOpts
	spinner := ysmrr.NewSpinner(opts)
	spinner.Error()

	var buf bytes.Buffer
	spinner.Print(&buf, "✗")

	want := fmt.Sprintf("%s %s\r\n", "✗", initialMessage)
	assert.Equal(t, want, buf.String())
}

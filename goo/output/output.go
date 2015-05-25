// Package output handles the output for Goophry.
// It provides routines to enable debug messages. Also output can be written to a
// logfile instead of printing it to stdout. Furthermore it provides the possibility
// to execute a command to notify an external script/application.
package output

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

var (
	outputCmdError       = "Error calling ErrorCmd:\n%s"
	outputCmdErrorOutput = "Error calling ErrorCmd:\n%s\n\nCommand:\n%s"
)

// outputLogger is an interface implemented by objects that
// handle the messages from within Goophry.
// This gives the possibility to provide our own handler for testing purposes.
type outputLogger interface {
	Println(...interface{})
}

// An Output provides routines to handle messages within Goophry.
type Output struct {
	debug     bool
	notifyCmd string
	logger    outputLogger
}

// NewOutput returns a new instance of Output, writing the messages to stdout per default.
func NewOutput() Output {
	l := log.New(os.Stdout, "", log.LstdFlags)

	return Output{
		logger: l,
	}
}

// SetDebug enables/disables debugging output.
func (o *Output) SetDebug(d bool) {
	o.debug = d
}

// SetNotifyCmd sets the command used for notifying about certain messages.
func (o *Output) SetNotifyCmd(cmd string) {
	o.notifyCmd = cmd
}

// SetLogfile modifies the Output object to write messages to the given logfile instead of stdout.
// It may return an error, if something went wrong with opening the file.
func (o *Output) SetLogfile(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	l := log.New(file, "", log.LstdFlags)
	o.logger = l
	return nil
}

// Debug writes a message to the current output, when debugging output is enabled.
func (o Output) Debug(msg string) {
	if o.debug {
		o.logger.Println(msg)
	}
}

// StopError writes a message to the current output, executes the notify-command
// and exits Goophry with the status 1.
func (o Output) StopError(msg string) {
	o.logger.Println(msg)
	o.notify(msg)
	os.Exit(1)
}

// NotifyError executes the notify-command with a given message.
func (o Output) NotifyError(msg string) {
	o.Debug(msg)
	o.notify(msg)
}

// notify tries to execute the notify-command with the given message.
// In case of an error it will write the error and message to the current output.
func (o Output) notify(msg string) {
	var err error
	var out []byte

	cmdExec := fmt.Sprintf(o.notifyCmd, strconv.Quote(msg))

	if o.notifyCmd == "" {
		return
	}

	out, err = exec.Command("sh", "-c", cmdExec).Output()

	if err != nil {
		o.logger.Println(fmt.Sprintf(outputCmdError, err))
	}

	if len(out) != 0 && err == nil {
		o.logger.Println(fmt.Sprintf(outputCmdErrorOutput, out, cmdExec))
	}
}

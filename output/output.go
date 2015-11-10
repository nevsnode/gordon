// Package output handles the output for Gordon.
// It provides routines to enable debug messages. Also output can be written to a
// logfile instead of printing it to stdout. Furthermore it provides the possibility
// to execute a command to notify an external script/application.
package output

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

var (
	outputTempFileError       = "Error writing temporary file: %s"
	outputTempFileRemoveError = "Error removing temporary file: %s"
	outputCmdError            = "Error calling ErrorScript: %s"
	outputCmdErrorOutput      = "Error calling ErrorScript (created output): %s; Script: %s"
)

// outputLogger is an interface implemented by objects that
// handle the messages from within Gordon.
// This gives the possibility to provide our own handler for testing purposes.
type outputLogger interface {
	Println(...interface{})
}

// An Output provides routines to handle messages within Gordon.
type Output struct {
	debug       bool
	errorScript string
	tempDir     string
	logger      outputLogger
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

// SetErrorScript sets the command used for notifying about certain messages.
func (o *Output) SetErrorScript(script string) {
	o.errorScript = script
}

// SetTempDir sets the directory used for temporary files.
func (o *Output) SetTempDir(tempDir string) {
	o.tempDir = tempDir
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
// and exits Gordon with the status 1.
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
	// Check if ErrorScript is even defined
	if o.errorScript == "" {
		return
	}

	// Write temporary file
	tempFile, err := o.writeTempFile(msg)
	if err != nil {
		o.logger.Println(fmt.Sprintf(outputTempFileError, err))

		// As the ErrorScript depends on the temporary file we stop here
		return
	}

	// Execute ErrorScript
	out, err := exec.Command(o.errorScript, tempFile).Output()

	// The ErrorScript caused an error ...
	if err != nil {
		o.logger.Println(fmt.Sprintf(outputCmdError, err))
	}

	// ... or returned output
	if len(out) != 0 && err == nil {
		o.logger.Println(fmt.Sprintf(outputCmdErrorOutput, out, o.errorScript))
	}

	// Remove temporary file
	err = os.Remove(tempFile)
	if err != nil {
		o.logger.Println(fmt.Sprintf(outputTempFileRemoveError, err))
	}
}

// writeTempFile creates a temporary file in the tempDir-directory that stores the given message.
func (o Output) writeTempFile(msg string) (filename string, err error) {
	file, err := ioutil.TempFile(o.tempDir, "gordon")
	if err != nil {
		return
	}

	_, err = file.WriteString(msg)
	if err != nil {
		return
	}

	filename = file.Name()
	return
}

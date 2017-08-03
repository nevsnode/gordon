// Package output handles the output for Gordon.
// It provides routines to enable debug messages. Also output can be written to a
// logfile instead of printing it to stdout. Furthermore it provides the possibility
// to execute a command to notify an external script/application.
package output

import (
	"fmt"
	"github.com/nevsnode/gordon/utils"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// outputLogger is an interface implemented by objects that
// handle the messages from within Gordon.
// This gives the possibility to provide our own handler for testing purposes.
type outputLogger interface {
	Println(...interface{})
}

const (
	prependDebug = "[DEBUG]"
	prependError = "[ERROR]"
)

var (
	debug       = false
	errorScript = ""
	tempDir     = ""
	logger      outputLogger

	errorOutputTempFile       = "Failed writing temporary file: %s"
	errorOutputTempFileRemove = "Failed removing temporary file: %s"
	errorOutputCmd            = "Failed calling ErrorScript: %s"
	errorOutputCmdOutput      = "Failed calling ErrorScript (created output): %s; Script: %s"
)

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

// SetDebug enables/disables debugging output.
func SetDebug(d bool) {
	debug = d
}

// SetErrorScript sets the command used for notifying about certain messages.
func SetErrorScript(script string) {
	errorScript = script
}

// SetTempDir sets the directory used for temporary files.
func SetTempDir(t string) {
	tempDir = t
}

// SetLogfile modifies the Output object to write messages to the given logfile instead of stdout.
// It may return an error, if something went wrong with opening the file.
func SetLogfile(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	logger = log.New(file, "", log.LstdFlags)
	return nil
}

func printLogger(msg ...interface{}) {
	text := strings.Replace(fmt.Sprintln(msg...), "\n", "\n\t", -1)
	logger.Println(strings.Trim(text, "\n\t"))
}

// Debug writes a message to the current output, when debugging output is enabled.
func Debug(msg ...interface{}) {
	if debug {
		printLogger(append([]interface{}{prependDebug}, msg...)...)
	}
}

// StopError writes a message to the current output, executes the notify-command
// and exits Gordon with the status 1.
func StopError(msg ...interface{}) {
	NotifyError(msg...)
	os.Exit(1)
}

// NotifyError executes the notify-command with a given message.
func NotifyError(msg ...interface{}) {
	printLogger(append([]interface{}{prependError}, msg...)...)
	notify(msg...)
}

// notify tries to execute the notify-command with the given message.
// In case of an error it will write the error and message to the current output.
func notify(msg ...interface{}) {
	// Check if error-script is even defined
	if errorScript == "" {
		return
	}

	// Write temporary file
	tempFile, err := writeTempFile(fmt.Sprintln(msg...))
	if err != nil {
		printLogger(fmt.Sprintf(errorOutputTempFile, err))

		// As the error-script depends on the temporary file we stop here
		return
	}

	// Execute error-script
	out, err := utils.ExecCommand(errorScript, tempFile).Output()

	// The error-script caused an error ...
	if err != nil {
		printLogger(fmt.Sprintf(errorOutputCmd, err))
	}

	// ... or returned output
	if len(out) != 0 && err == nil {
		printLogger(fmt.Sprintf(errorOutputCmdOutput, out, errorScript))
	}

	// Remove temporary file
	err = os.Remove(tempFile)
	if err != nil {
		printLogger(fmt.Sprintf(errorOutputTempFileRemove, err))
	}
}

// writeTempFile creates a temporary file in the tempDir-directory that stores the given message.
func writeTempFile(msg string) (filename string, err error) {
	file, err := ioutil.TempFile(tempDir, "gordon")
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

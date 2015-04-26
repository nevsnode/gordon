package output

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

var (
	emptyCmdError        = fmt.Errorf("ErrorCmd is empty/not set.")
	outputCmdError       = "Error calling ErrorCmd:\n%s"
	outputCmdErrorOutput = "Error calling ErrorCmd:\n%s\n\nCommand:\n%s"
)

type outputLogger interface {
	Println(...interface{})
}

type Output struct {
	debug     bool
	notifyCmd string
	logger    outputLogger
}

func New() Output {
	l := log.New(os.Stdout, "", log.LstdFlags)

	return Output{
		logger: l,
	}
}

func (o *Output) SetDebug(d bool) {
	o.debug = d
}

func (o *Output) SetNotifyCmd(cmd string) {
	o.notifyCmd = cmd
}

func (o Output) Debug(msg string) {
	if o.debug {
		o.logger.Println(msg)
	}
}

func (o Output) StopError(msg string) {
	o.logger.Println(msg)
	o.notify(msg)
	os.Exit(1)
}

func (o Output) NotifyError(msg string) {
	o.notify(msg)
}

func (o Output) notify(msg string) {
	var err error
	var out []byte

	cmdExec := fmt.Sprintf(o.notifyCmd, strconv.Quote(msg))

	if o.notifyCmd == "" {
		err = emptyCmdError
	} else {
		out, err = exec.Command("sh", "-c", cmdExec).Output()
	}

	if err != nil {
		o.logger.Println(fmt.Sprintf(outputCmdError, err))
	}

	if len(out) != 0 && err == nil {
		o.logger.Println(fmt.Sprintf(outputCmdErrorOutput, out, cmdExec))
	}
}

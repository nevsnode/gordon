package output

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
)

var errorCmdEmpty = fmt.Errorf("ErrorCmd is empty/not set.")

type Output struct {
	debug     bool
	notifyCmd string
}

func New() Output {
	return Output{}
}

func (o *Output) SetDebug(d bool) {
	o.debug = d
}

func (o *Output) SetNotifyCmd(cmd string) {
	o.notifyCmd = cmd
}

func (o Output) Debug(msg string) {
	if o.debug {
		log.Println(msg)
	}
}

func (o Output) StopError(msg string) {
	o.notify(msg)
	log.Fatal(msg)
}

func (o Output) NotifyError(msg string) {
	o.notify(msg)
}

func (o Output) notify(msg string) {
	var err error
	var out []byte

	cmdExec := fmt.Sprintf(o.notifyCmd, strconv.Quote(msg))

	if o.notifyCmd == "" {
		err = errorCmdEmpty
	} else {
		out, err = exec.Command("sh", "-c", cmdExec).Output()
	}

	if err != nil {
		log.Println(fmt.Sprintf("Error calling ErrorCmd:\n%s", err))
	}

	if len(out) != 0 && err == nil {
		log.Println(fmt.Sprintf("Error calling ErrorCmd:\n%s\n\nCommand:\n%s", out, cmdExec))
	}
}

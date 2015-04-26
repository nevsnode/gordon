package main

import (
	"./goo/basepath"
	"./goo/config"
	"./goo/output"
	"./goo/taskqueue"
	"flag"
	"fmt"
	"log"
)

var (
	configFile string
	verbose    bool
)

func init() {
	flag.StringVar(&configFile, "c", "", "path to config file")
	flag.BoolVar(&verbose, "v", false, "enable verbose/debugging output")
}

func main() {
	flag.Parse()

	base, err := basepath.New()
	if err != nil {
		log.Fatal("basepath: ", err)
	}

	if configFile == "" {
		configFile = base.GetPathWith("./goophry.config.json")
	}

	conf, err := config.New(configFile)
	if err != nil {
		log.Fatal("config: ", err)
	}

	out := output.New()
	out.SetDebug(verbose)
	out.SetNotifyCmd(conf.ErrorCmd)

	tq := taskqueue.New()
	tq.SetConfig(conf)
	tq.SetOutput(out)

	for _, ct := range conf.Tasks {
		queue := make(chan taskqueue.QueueTask)

		ct.Script = base.GetPathWith(ct.Script)

		if ct.Workers <= 1 {
			ct.Workers = 1
		}

		for i := 0; i < ct.Workers; i++ {
			tq.WaitGroup.Add(1)
			go tq.TaskWorker(ct, queue)
		}
		out.Debug(fmt.Sprintf("Created %d workers for type %s", ct.Workers, ct.Type))

		tq.WaitGroup.Add(1)
		go tq.QueueWorker(ct, queue)
		out.Debug(fmt.Sprintf("Created queue worker for type %s", ct.Type))
	}

	tq.WaitGroup.Wait()
}
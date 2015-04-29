// This is the Goophry main application.
// It parses all commandline flags, creates instances of all necessary packages
// and triggers the creation of all worker go-routines.
package main

import (
	"./goo/basepath"
	"./goo/config"
	"./goo/output"
	"./goo/stats"
	"./goo/taskqueue"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	configFile string
	verbose    bool
	logfile    string
)

func init() {
	flag.StringVar(&configFile, "c", "", "path to config file")
	flag.StringVar(&logfile, "l", "", "path to logfile")
	flag.BoolVar(&verbose, "v", false, "enable verbose/debugging output")
}

func main() {
	flag.Parse()

	base, err := basepath.NewBasepath()
	if err != nil {
		log.Fatal("basepath: ", err)
	}

	// When no configuration file was passed as a flag, use the default location.
	if configFile == "" {
		configFile = base.GetPathWith("./goophry.config.json")
	}

	conf, err := config.NewConfig(configFile)
	if err != nil {
		log.Fatal("config: ", err)
	}

	out := output.NewOutput()
	out.SetDebug(verbose)
	out.SetNotifyCmd(conf.ErrorCmd)

	// When no logfile was passed as a flag but it was set in the configuration,
	// use that one instead.
	if logfile == "" && conf.Logfile != "" {
		logfile = base.GetPathWith(conf.Logfile)
	}

	if logfile != "" {
		err = out.SetLogfile(logfile)

		if err != nil {
			log.Fatal("out.SetLogfile(): ", err)
		}
	}

	sta := stats.NewStats()

	tq := taskqueue.NewTaskqueue()
	tq.SetConfig(conf)
	tq.SetOutput(out)
	tq.SetStats(&sta)

	for _, ct := range conf.Tasks {
		ct.Script = base.GetPathWith(ct.Script)
		tq.CreateWorkers(ct)

		sta.InitTask(ct.Type)
	}

	// If the StatsInterface was set, start the HTTP-server for it.
	if conf.StatsInterface != "" {
		go func() {
			out.Debug("Serving stats on http://" + conf.StatsInterface)
			err := sta.ServeHttp(conf.StatsInterface)

			if err != nil {
				out.NotifyError(fmt.Sprintf("stats.ServeHttp(): %s", err))
			}
		}()
	}

	// Start another go-routine to initiate the graceful shutdown of all taskqueue-workers,
	// when the application shall be terminated.
	cc := make(chan os.Signal)
	signal.Notify(cc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-cc
		tq.Stop()
	}()

	tq.Wait()
}

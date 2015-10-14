// This is the Gordon main application.
// It parses all commandline flags, creates instances of all necessary packages
// and triggers the creation of all worker go-routines.
package main

import (
	"./go/basepath"
	"./go/config"
	"./go/output"
	"./go/stats"
	"./go/taskqueue"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const GordonVersion = "1.3.2"

var (
	configFile  string
	verbose     bool
	logfile     string
	showVersion bool
)

func init() {
	flag.StringVar(&configFile, "c", "", "path to config file")
	flag.StringVar(&logfile, "l", "", "path to logfile")
	flag.BoolVar(&verbose, "v", false, "enable verbose/debugging output")
	flag.BoolVar(&showVersion, "V", false, "show version")
}

func main() {
	flag.Parse()

	if showVersion == true {
		fmt.Printf("Gordon version %s\n", GordonVersion)
		os.Exit(0)
	}

	base, err := basepath.NewBasepath()
	if err != nil {
		log.Fatal("basepath: ", err)
	}

	// When no configuration file was passed as a flag, use the default location.
	if configFile == "" {
		configFile = base.GetPathWith("./gordon.config.json")
	}

	conf, err := config.NewConfig(configFile)
	if err != nil {
		log.Fatal("config: ", err)
	}

	out := output.NewOutput()
	out.SetDebug(verbose)
	out.SetErrorScript(conf.ErrorScript)
	out.SetTempDir(base.GetPathWith(conf.TempDir))

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
	sta.SetVersion(GordonVersion)

	tq := taskqueue.NewTaskqueue()
	tq.SetOutput(out)
	tq.SetConfig(conf)
	tq.SetStats(&sta)

	for _, ct := range conf.Tasks {
		ct.Script = base.GetPathWith(ct.Script)
		tq.CreateWorkers(ct)

		sta.InitTask(ct.Type)
	}

	// If the StatsInterface was set, start the HTTP-server for it.
	if conf.StatsInterface != "" {
		if conf.StatsPattern == "" {
			conf.StatsPattern = "/"
		}

		go func() {
			var err error

			if conf.StatsTLSCertFile != "" {
				conf.StatsTLSCertFile = base.GetPathWith(conf.StatsTLSCertFile)
				conf.StatsTLSKeyFile = base.GetPathWith(conf.StatsTLSKeyFile)

				out.Debug("Serving stats on https://" + conf.StatsInterface + conf.StatsPattern)
				err = sta.ServeHttps(conf.StatsInterface, conf.StatsPattern, conf.StatsTLSCertFile, conf.StatsTLSKeyFile)
			} else {
				out.Debug("Serving stats on http://" + conf.StatsInterface + conf.StatsPattern)
				err = sta.ServeHttp(conf.StatsInterface, conf.StatsPattern)
			}

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

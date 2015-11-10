// This is the Gordon main application.
// It parses all commandline flags, creates instances of all necessary packages
// and triggers the creation of all worker go-routines.
package main

import (
	"./basepath"
	"./config"
	"./output"
	"./stats"
	"./taskqueue"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	GordonVersion = "1.4.0"
	defaultConfig = "./gordon.config.toml"
)

var cli struct {
	Config  string
	Test    bool
	Verbose bool
	Version bool
}

func init() {
	flag.StringVar(&cli.Config, "c", "", "path to config file")
	flag.BoolVar(&cli.Test, "t", false, "test configuration file")
	flag.BoolVar(&cli.Verbose, "v", false, "enable verbose/debugging output")
	flag.BoolVar(&cli.Version, "V", false, "show version")
}

func main() {
	flag.Parse()

	if cli.Version == true {
		fmt.Printf("Gordon version %s\n", GordonVersion)
		os.Exit(0)
	}

	base, err := basepath.NewBasepath()
	if err != nil {
		log.Fatal("basepath: ", err)
	}

	// When no configuration file was passed as a flag, use the default location.
	if cli.Config == "" {
		cli.Config = base.GetPathWith(defaultConfig)
	}

	conf, err := config.NewConfig(cli.Config)

	// When test-flag is set, respond accordingly
	if cli.Test {
		if err != nil {
			fmt.Println("Configuration is invalid: ", err)
		} else {
			fmt.Println("Configuration is valid")
		}
		os.Exit(0)
	}

	if err != nil {
		log.Fatal("config: ", err)
	}

	out := output.NewOutput()
	out.SetDebug(cli.Verbose)
	out.SetErrorScript(conf.ErrorScript)
	out.SetTempDir(base.GetPathWith(conf.TempDir))

	// Set logfile for output, when configured
	if conf.Logfile != "" {
		err = out.SetLogfile(base.GetPathWith(conf.Logfile))

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
	if conf.Stats.Interface != "" {
		if conf.Stats.Pattern == "" {
			conf.Stats.Pattern = "/"
		}

		go func() {
			var err error

			if conf.Stats.TLSCertFile != "" {
				conf.Stats.TLSCertFile = base.GetPathWith(conf.Stats.TLSCertFile)
				conf.Stats.TLSKeyFile = base.GetPathWith(conf.Stats.TLSKeyFile)

				out.Debug("Serving stats on https://" + conf.Stats.Interface + conf.Stats.Pattern)
				err = sta.ServeHttps(conf.Stats.Interface, conf.Stats.Pattern, conf.Stats.TLSCertFile, conf.Stats.TLSKeyFile)
			} else {
				out.Debug("Serving stats on http://" + conf.Stats.Interface + conf.Stats.Pattern)
				err = sta.ServeHttp(conf.Stats.Interface, conf.Stats.Pattern)
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

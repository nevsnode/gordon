package main

import (
	"flag"
	"fmt"
	"github.com/nevsnode/gordon/config"
	"github.com/nevsnode/gordon/output"
	"github.com/nevsnode/gordon/stats"
	"github.com/nevsnode/gordon/taskqueue"
	"github.com/nevsnode/gordon/utils"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// GordonVersion is the current version of Gordon
const GordonVersion = "1.6.5"

const cliDefaultLogfile = "-"

var cli struct {
	config      string
	configLong  string
	logfile     string
	test        bool
	testLong    bool
	verbose     bool
	verboseLong bool
	version     bool
}

func init() {
	flag.StringVar(&cli.config, "c", "", "path to config file")
	flag.StringVar(&cli.configLong, "conf", "", "path to config file")
	flag.StringVar(&cli.logfile, "logfile", cliDefaultLogfile, "path to logfile (overwrites configuration)")
	flag.BoolVar(&cli.test, "t", false, "test configuration file")
	flag.BoolVar(&cli.testLong, "test", false, "test configuration file")
	flag.BoolVar(&cli.verbose, "v", false, "enable verbose/debugging output")
	flag.BoolVar(&cli.verboseLong, "verbose", false, "enable verbose/debugging output")
	flag.BoolVar(&cli.version, "version", false, "show version")
}

func main() {
	flag.Parse()

	if cli.configLong != "" {
		cli.config = cli.configLong
	}
	if cli.testLong {
		cli.test = true
	}
	if cli.verboseLong {
		cli.verbose = true
	}

	if cli.version == true {
		fmt.Printf("Gordon version %s\n", GordonVersion)
		os.Exit(0)
	}

	// When no configuration file was passed as a flag, use the default location.
	if cli.config == "" {
		cli.config = utils.Basepath(config.DefaultConfig)
	}

	conf, err := config.New(cli.config)

	// When test-flag is set, respond accordingly
	if cli.test {
		if err != nil {
			fmt.Println("Configuration is invalid:", err)
		} else {
			fmt.Println("Configuration is valid")
		}
		os.Exit(0)
	}

	if err != nil {
		log.Fatal("Configuration is invalid:", err)
	}

	// Overwrite logfile, if value is passed as a flag
	if cli.logfile != cliDefaultLogfile {
		conf.Logfile = cli.logfile
	}

	stats.GordonVersion = GordonVersion
	output.SetDebug(cli.verbose)
	output.SetErrorScript(conf.ErrorScript)
	output.SetTempDir(utils.Basepath(conf.TempDir))

	// Set logfile for output, when configured
	if conf.Logfile != "" {
		err = output.SetLogfile(utils.Basepath(conf.Logfile))
		if err != nil {
			log.Fatal("output.SetLogfile(): ", err)
		}
	}

	stats.Setup(conf.Stats)
	taskqueue.Start(conf)

	// Start another go-routine to initiate the graceful shutdown of all taskqueue-workers,
	// when the application shall be terminated.
	cc := make(chan os.Signal)
	signal.Notify(cc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-cc
		output.Debug("Stopping taskqueue")
		taskqueue.Stop()
	}()

	output.Debug("Up and waiting for tasks")
	taskqueue.Wait()
}

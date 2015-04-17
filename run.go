package main

import (
    "fmt"
    "flag"
    "os"
    "os/exec"
    "encoding/json"
    "path/filepath"
    "./lockfile"
    "strings"
)

type configValues struct {
    RedisHost string
    RedisPort int
    RedisQueueKey string
    Worker int
    Tasks map[string]string
    Lockfile string
    ErrorCmd string
}

type task {
    Type string
    Args []string
}

var (
    config configValues
    verbose bool
    lock lockfile.Lockfile
    basepath string
)

func init() {
    flag.BoolVar(&verbose, "v", false, "enable verbose/debugging output")
	flag.Parse()

    basepath, err := getBasePath()
    if err != nil {
        log.Fatal("getBasePath(): ", err)
    }

    config, err := getConfig()
    if err != nil {
        log.Fatal("getConfig(): ", err)
    }

    if filepath.isAbs(config.Lockfile) == false {
        config.Lockfile = basepath + "/" + config.Lockfile
    }
    lock, err := lockfile.New(config.Lockfile)
    if err != nil {
        log.Fatal(err)
    }
    err = lock.Create()
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    // TODO create workers

    // TODO subscribe to redis queue
}

func debugOutput(msg string) {
	if verbose == true {
		log.Println(msg)
	}
}

func getBasePath() (path string, err error) {
    path, err = filepath.Abs(filepath.Dir(os.Args[0]))
    return
}

func getConfig() (c configValues, err error) {
	file, err := os.Open(basepath + "/config.json")
	if err != nil {
		return
	}
	defer file.Close()

	parser := json.NewDecoder(file)
	err = parser.Decode(&c)
	return
}

func exit() {
    // TODO close connection?

    // TODO remove lock file
    err = lock.Remove()
    if err != nil {
        log.Fatal(err)
    }
}

func handleTask(t task) {
    // TODO check if config.Tasks[t.Type] is defined
    file := ""

    // TODO execute task and on failure execute errorcmd with output
    err := executeTask(file, t.Args)
    if err != nil {
        errorCmd(fmt.Sprintf("%s %s\n\n%s", file, strings.Join(t.Args, " "), err))
    }
}

func errorCmd(msg string) {
    debugOutput(fmt.Sprintf("Calling ErrorCmd with: %s", msg))

    cmd := fmt.Sprintf(config.ErrorCmd, msg)
    out, err := exec.Command("sh", "-c", cmd).Output()
    if len(out) != 0 && err == nil {
        err = fmt.Sprintf("%s\n\nOutput:\n%s", cmd, out)
    }
    if err != nil {
        log.Println(fmt.Sprintf("Error calling ErrorCmd:\n%s", err))
    }
}

func executeTask(file string, args []string) (error) {
    out, err := exec.Command(file, args...).Output()
    if len(out) != 0 && err == nil {
        err = fmt.Errorf("%s", out)
    }
    return err
}

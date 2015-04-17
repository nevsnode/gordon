package lockfile

import (
    "path/filepath"
    "os"
    "fmt"
)

type Lockfile struct {
    path string
}

func New(p string) (Lockfile, error) {
    if filepath.IsAbs(p) == false {
        return Lockfile{path: ""}, fmt.Errorf("The Lockfile must be an absolute path")
    }
    return Lockfile{path: p}, nil
}

func (l Lockfile) Exists() (bool) {
    file, err := os.Open(l.path)
    defer file.Close()

    if err == nil {
        return true
    }

    if os.IsNotExist(err) {
        return false
    }

    return true
}

func (l Lockfile) Create() (error) {
    file, err := os.Create(l.path)
    defer file.Close()
    return err
}

func (l Lockfile) Remove() (error) {
    return os.Remove(l.path)
}

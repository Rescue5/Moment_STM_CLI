//go:build !windows
package dmsx

import (
    "os"
    "syscall"
    "golang.org/x/term"
)

type portType = *os.File

func (dev *device) open() error {
    if f, err := os.OpenFile(dev.dsn, syscall.O_RDWR | syscall.O_NOCTTY, 0644); err != nil {
        return err
    } else if !term.IsTerminal(int(f.Fd())) {
        return errorf("%s - not a terminal", dev.dsn)
    } else if _, err := term.MakeRaw(int(f.Fd())); err != nil {
        return err
    } else {
        dev.file = f
        return nil
    }
}
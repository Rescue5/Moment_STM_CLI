//go:build windows
package dmsx

import (
	"github.com/albenik/go-serial/v2"
)

type portType struct {
	*serial.Port
}

func (dev *device) open() error {
	options := []serial.Option{
		serial.WithReadTimeout(1),
		serial.WithBaudrate(115200),
	}

	if f, err := serial.Open(dev.dsn, options...); err != nil {
		return err
	} else {
		f.SetDTR(true)
		dev.file = portType{ f }
		return nil
	}
}
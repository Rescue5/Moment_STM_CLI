package dmsx

import (
	"fmt"
	"sync"
	"time"
	"bufio"
	"bytes"

	"context"

	"github.com/sourcegraph/conc"
	"github.com/albenik/go-serial"

	. "dronmotors/dmetrics/pkg/helpers"
	. "dronmotors/dmetrics/internal/device"

	dms "dronmotors/dmetrics/internal/script"
)

func errorf(t string, args ...interface{}) error{
	return fmt.Errorf("dmsx: " + t, args...)
}

type device struct {
	sync.Mutex
	conc.WaitGroup
	callbacks Callbacks

	id string
	dsn string
	status int
	lastText string

	sp serial.Port
	cancel context.CancelCauseFunc
	controlMtx sync.Mutex
}

func NewDevice(dsn string, callbacks Callbacks) Device {
	return &device{
		dsn: dsn,
		lastText: "?",
		callbacks: callbacks,
		status: StatusDisconnected,
	}
}

func (dev device) Id() string {
	return dev.id
}

func (dev device) Status() string {
	switch dev.status {
	case StatusConnected:
		return "connected"
	case StatusDisconnected:
		return "disconnected"
	default:
		panic(1)
	}
}

func (dev device) Methods() []string {
	return []string{
		"id",
		"tare",
		"brake",
		"sample",
		"chiller",
		"throttle",
	}
}

func (dev *device) open() error {
	mode := &serial.Mode{}
}

	if sp, err := serial.Open(dev.dsn, mode); err != nil {
		return err
	} else {
		dev.sp = sp
	}

	return nil
}

func (dev *device) close() error {
	return dev.sp.Close()
}



func cmdf(t string, args ...interface{}) string {
	return fmt.Sprintf("/" + t + "\n", args...)
}

func (dev *device) ping() {
	dev.control(cmdf("ping"), 0)
}

func (dev *device) control(cmd string, deadline time.Duration) (interface{}, error) {
	dev.controlMtx.Lock()
	defer dev.controlMtx.Unlock()

	dev.lastText = "?" // reset before sending command
	if n, err := dev.sp.Write([]byte(cmd)); err != nil {
		return nil, err
	} else if n != len(cmd) {
		return nil, errorf("control write error, fix the code")
	} else if deadline == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, errorf("control timeout")
		case <-time.After(1 * time.Millisecond):
			if len(dev.lastText) > 1 {
				return dev.lastText, nil
			}
		}
	}
}

func (dev *device) Control(cmd string, args ...dms.Value) (res interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if v, ok := r.(error); ok {
				err = v
			} else {
				err = errorf("unknown error in control")
			}
		}
	}()

	deadline := 1000 * time.Millisecond

	switch cmd {
	case "id":
		if len(dev.id) > 0 {
			return dev.id, nil
		} else {
			return dev.control(cmdf("id"), deadline)
		}
	case "tare":
		return dev.control(cmdf("tare"), deadline)
	case "brake":
		return dev.control(cmdf("brake=%d,%d", args[0].Int(), args[1].Int()), deadline)
	case "sample":
		return dev.control(cmdf("sample=%d", args[0].Int()), deadline)
	case "chiller":
		return dev.control(cmdf("chiller=%d,%d", args[0].Int(), args[1].Int()), deadline)
	case "throttle":
		return dev.control(cmdf("throttle=%d", args[0].Int()), deadline)
	}

	return nil, errorf("no such control command")
}

func (dev *device) connect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if v, ok := r.(error); ok {
				err = v
			} else {
				err = errorf("unknown error in connect callback")
			}
		}
	}()

	dev.Lock()
	defer dev.Unlock()

	if dev.status == StatusDisconnected {
		dev.callbacks.OnConnect(dev)
		dev.status = StatusConnected
	}

	return
}

func (dev *device) telemetry(t Telemetry) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if v, ok := r.(error); ok {
				err = v
			} else {
				err = errorf("unknown error in telemetry callback")
			}
		}
	}()

	if dev.status == StatusConnected {
		dev.callbacks.OnTelemetry(dev, t)
	}

	return
}

func (dev *device) disconnect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if v, ok := r.(error); ok {
				err = v
			} else {
				err = errorf("unknown error in disconnect callback")
			}
		}
	}()

	dev.Lock()
	defer dev.Unlock()

	if dev.status == StatusConnected {
		dev.status = StatusDisconnected
		dev.callbacks.OnDisconnect(dev)
	}

	return
}

func (dev *device) process(parentCtx context.Context) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	dev.Go(func() {
		defer cancel()
		for ctx.Err() == nil {
			dev.ping()
			select {
			case <-ctx.Done():
			case <-time.After(100 * time.Millisecond):
			}
		}
	})

	//
	// STREAM: ... | SYN | CHL | ... data ... | CRC | FIN | ...
	// TONKEN:           < CHL | ... data ... | CRC >
	//

	splitter := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		var syn []byte = []byte{ 0xc1, 0xc1, 0xc1, 0xc1 }
		var fin []byte = []byte{ 0xc2, 0xc2, 0xc2, 0xc2 }

		syn_idx := bytes.Index(data, syn)
		fin_idx := bytes.Index(data, fin)

		if syn_idx >= 0 && (fin_idx > syn_idx) {
			return fin_idx + 4, data[syn_idx + 4 : fin_idx], nil // cut off syn & fin
		}

		const overrun int = 256 // avoid reader buffer overrun

		if atEOF {
			return len(data), nil, bufio.ErrFinalToken
		} else if syn_idx < 0 || len(data) > overrun {
			return 1, nil, nil
		} else {
			return syn_idx, nil, nil
		}
	}

	scanner := bufio.NewScanner(ContextualReader(ctx, dev.sp))
	scanner.Split(splitter)

	// TODO: scanner timeout

	for scanner.Scan() {
		if t := scanner.Text(); len(t) > 0 {
			if f, err := decodeFrame([]byte(t)); err != nil {
				fmt.Println(err)
			} else {
				switch f.Channel {
				case frameChannelText:
					dev.lastText = string(f.Payload)
				case frameChannelData:
					d, err := dataTelemetry{}.decode(f)
					if err != nil {
						fmt.Println(err)
					} else if err := dev.telemetry(d); err != nil {
						fmt.Println(err)
					}
				}
			}
		}
	}

	return nil
}

func (dev *device) identify(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if v, err := dev.Control("id"); err == nil {
			dev.id = v.(string)
			return nil
		} else if time.Now().After(deadline) {
			return err
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func (dev *device) StartUp(parentCtx context.Context) error {
	ctx, cancel := context.WithCancelCause(parentCtx)
	if err := dev.open(); err != nil {
		return err
	} else {
		dev.cancel = cancel

		dev.Go(func() {
			defer func() {
				dev.close()
				if err := dev.disconnect(); err != nil {
					fmt.Println(err)
				}
			}()
			cancel(dev.process(ctx))
		})

		const identifyTimeout = 1000 * time.Millisecond
		if err := dev.identify(ctx, identifyTimeout); err != nil {
			defer dev.Wait()
			cancel(err)
		} else if err := dev.connect(); err != nil {
			defer dev.Wait()
			cancel(err)
		}
		
		return context.Cause(ctx)
	}
}

func (dev *device) TearDown() error {
	defer dev.Wait()
	dev.cancel(nil)
	return nil
}

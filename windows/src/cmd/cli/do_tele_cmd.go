package main

import (
	"fmt"
	"context"

	"github.com/urfave/cli/v2"

	"dronmotors/dmetrics/internal/device"
	"dronmotors/dmetrics/internal/device/dmsx"

	dms "dronmotors/dmetrics/internal/script"
)

func (app *App) doTeleCmd(cli *cli.Context) error {
	ctx, cancel := context.WithCancelCause(cli.Context)
	defer cancel(nil)

	callbacks := &device.CallbacksWrapper{
		Connect: func(dev device.Device) {
			dev.Control("sample", dms.NewValue(cli.Int("rate")))
			app.Go(func() {
				select {
				case <-ctx.Done():
					break
				}
			})
		},
		Telemetry: func(dev device.Device, t device.Telemetry) {
			fmt.Println(t)
		},
		Disconnect: func(dev device.Device) {
			cancel(errorf("disconnected"))
		},
	}

	dev := dmsx.NewDevice(cli.String("port"), callbacks)
	if err := dev.StartUp(ctx); err != nil {
		return err
	} else {
		defer dev.TearDown()
	}

	app.Wait()

	return context.Cause(ctx)
}

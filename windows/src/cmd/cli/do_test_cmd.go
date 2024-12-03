package main

import (
	"os"
	"context"

	"github.com/urfave/cli/v2"

	"dronmotors/dmetrics/internal/device"
	"dronmotors/dmetrics/internal/device/dmsx"

	"dronmotors/dmetrics/internal/script/lua"
)

func (app *App) doTestCmd(cli *cli.Context) error {
	ctx, cancel := context.WithCancelCause(cli.Context)
	defer cancel(nil)

	filename := "default.lua"
	if cli.Args().Present() {
		filename = cli.Args().First()
	}

	filedata, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	ls, err := lua.NewScript(string(filedata), app.argsMap(cli))
	if err != nil {
		return err
	} else {
		defer ls.Release()
	}

	callbacks := &device.CallbacksWrapper{
		Connect: func(dev device.Device) {
			if err := ls.Execute(context.Background(), "OnConnect"); err != nil {
				panic(err)
			} else {
				app.Go(func() {
					defer ls.Execute(context.Background(), "OnDisconnect")
					cancel(ls.Execute(ctx, "Test"))
				})
			}
		},
		Telemetry: func(dev device.Device, t device.Telemetry) {
			if err := ls.Execute(ctx, "OnTelemetry", t); err != nil {
				cancel(err)
			} else {
				app.telemetry = append(app.telemetry, t)
			}
		},
		Disconnect: func(dev device.Device) {
			cancel(errorf("disconnected"))
		},
	}

	dev := dmsx.NewDevice(cli.String("port"), callbacks)

	if res, err := ls.Bind(dev); err != nil {
		return err
	} else {
		defer res.Release()
	}

	if err := dev.StartUp(ctx); err != nil {
		return err
	} else {
		defer dev.TearDown()
	}

	app.Wait()

	return context.Cause(ctx)
}

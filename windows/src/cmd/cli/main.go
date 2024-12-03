package main

import (
	"os"
	"fmt"
	"context"
	"strings"
	"syscall"
	"os/signal"

	"github.com/urfave/cli/v2"
	"github.com/sourcegraph/conc"

	"dronmotors/dmetrics/internal/device"

	"encoding/csv"
)

////////////////////////////////////////////////////////////////////////////////

func errorf(t string, args ...interface{}) error {
	return fmt.Errorf(t, args...)
}

////////////////////////////////////////////////////////////////////////////////

type App struct {
        *cli.App
        conc.WaitGroup
	telemetry []device.Telemetry
}

func (app *App) argsMap(cli *cli.Context) map[string]string {
	m := map[string]string{}

	for _, arg := range cli.StringSlice("args") {
		v := strings.Split(arg, "=")
		if len(v) == 1 {
			m[ v[0] ] = ""
		} else if len(v) == 2 {
			m[ v[0] ] = v[1]
		}
	}

	return m
}

func (app *App) SaveTelemetry(filename string) error {
	if len(app.telemetry) == 0 {
		return nil
	}

	if f, err := os.Create(filename); err != nil {
		return err
	} else {
		defer f.Close()

                writer := csv.NewWriter(f)
                defer writer.Flush()
		
		var data [][]string
		data = append(data, append([]string{ "idx" }, app.telemetry[0].AsKeys()...))
		for i, t := range app.telemetry {
			data = append(data, append([]string{ fmt.Sprintf("%d", i) }, t.AsValues()... ))
		}

		return writer.WriteAll(data)
	}
}

func NewApp() *App {
	app := &App{}

	app.App = &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "test",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "port",
						Usage: "port to use",
						Value: "/dev/tty.usbmodem101",
					},
					&cli.StringSliceFlag{
						Name: "args",
						Usage: "args to pass to the script",
					},
				},
				Action: func(cli *cli.Context) error {
					return app.doTestCmd(cli)
				},
			},
			{
				Name:  "tele",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name: "rate",
						Usage: "sample rate, ms",
						Value: 10,
					},
					&cli.StringFlag{
						Name: "port",
						Usage: "port to use",
						Value: "/dev/tty.usbmodem101",
					},
				},
				Action: func(cli *cli.Context) error {
					return app.doTeleCmd(cli)
				},
			},
			{
				Name:  "repl",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "port",
						Usage: "port to use",
						Value: "/dev/tty.usbmodem101",
					},
					&cli.StringFlag{
						Name: "tele",
						Usage: "tele file to use",
						Value: "telemetry.bin",
					},
				},
				Action: func(cli *cli.Context) error {
					return app.doReplCmd(cli)
				},
			},
		},
	}
	return app
}

func main() {
        app := NewApp()

        signals := []os.Signal{
                syscall.SIGHUP,
                syscall.SIGINT,
                syscall.SIGTERM,
                syscall.SIGQUIT,
        }

        ctx, cancel := signal.NotifyContext(context.Background(), signals...)
        defer cancel()

        if err := app.RunContext(ctx, os.Args); err != nil {
                if err != context.Canceled {
			fmt.Println(err)
		} else {
			if err := app.SaveTelemetry("telemetry.csv"); err != nil {
				fmt.Println(err)
			}
		}
	}
}

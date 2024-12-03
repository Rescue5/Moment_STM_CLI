package main

import (
	"os"
	"fmt"
	"context"
	"strings"

	"github.com/urfave/cli/v2"

	"dronmotors/dmetrics/internal/device"
	"dronmotors/dmetrics/internal/device/dmsx"

	dms "dronmotors/dmetrics/internal/script"

	"github.com/chzyer/readline"
)

func (app *App) doReplCmd(cli *cli.Context) error {
	ctx, cancel := context.WithCancelCause(cli.Context)
	defer cancel(nil)

	stdin := readline.NewCancelableStdin(os.Stdin)
	rlcfg := &readline.Config{
		Stdin: stdin,
	}

	rl, err := readline.NewEx(rlcfg)
	if err != nil {
		return err
	}

	defer rl.Close()

	telefile, err := os.Create(cli.String("tele"))
	if err != nil {
		return err
	}

	defer telefile.Close()

	callbacks := &device.CallbacksWrapper{
		Connect: func(dev device.Device) {
			methods := []readline.PrefixCompleterInterface{}
			for _, method := range dev.Methods() {
				methods = append(methods, readline.PcItem(method))
			}

			methods = append(methods, readline.PcItem("/quit"))
			completer := readline.NewPrefixCompleter(
				methods...
			)

			rlcfg.AutoComplete = completer
			rl.SetPrompt(fmt.Sprintf("[%s]> ", dev.Id()))

			app.Go(func() {
				defer cancel(nil)

				for {
					line, err := rl.Readline()
					line = strings.TrimSpace(line)

					if err != nil {
						cancel(err)
						break
					} else if line == "/quit" || line == "/exit" {
						break
					} else if args := strings.Fields(line); len(args) >= 1 {
						cmd_args := []dms.Value{}
						for _, v := range args[1:] {
							cmd_args = append(cmd_args, dms.NewValue(v))
						}

						if res, err := dev.Control(args[0], cmd_args...); err != nil {
							fmt.Println(err)
						} else {
							fmt.Println(res)
						}
					}
				}
			})
		},
		Telemetry: func(dev device.Device, t device.Telemetry) {
			fmt.Fprintf(telefile, "%s\n", t)
		},
		Disconnect: func(dev device.Device) {
			stdin.Close()
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

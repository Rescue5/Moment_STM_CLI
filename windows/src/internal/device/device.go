package device

import (
	"context"

	dms "dronmotors/dmetrics/internal/script"
)

type Device interface {
	Id() string
	Status() string
	StartUp(context.Context) error
	TearDown() error
	// scriptable
	Control(cmd string, args ...dms.Value) (interface{}, error)
	Methods() []string
}

const (
	StatusConnected		= iota
	StatusDisconnected
)

type Callbacks interface {
	OnConnect(Device)
	OnTelemetry(Device, Telemetry)
	OnDisconnect(Device)
}

type CallbacksWrapper struct {
	Callbacks

	Connect func(Device)
	Telemetry func(Device, Telemetry)
	Disconnect func(Device)
}

func (p CallbacksWrapper) OnConnect(dev Device) {
	p.Connect(dev)
}

func (p CallbacksWrapper) OnTelemetry(dev Device, t Telemetry) {
	p.Telemetry(dev, t)
}

func (p CallbacksWrapper) OnDisconnect(dev Device) {
	p.Disconnect(dev)
}

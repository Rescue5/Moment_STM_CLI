package device

import (
	"time"
)

type Telemetry interface {
	Id() string

	AsKeys() []string
	AsValues() []string
	TimeStamp() time.Time

	String() string
}

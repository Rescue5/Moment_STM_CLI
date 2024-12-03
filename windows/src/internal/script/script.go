package script

import (
	"fmt"
	"context"
	"strconv"
)

////////////////////////////////////////////////////////////////////////////////

func errorf(t string, args ...interface{}) error {
	return fmt.Errorf("script: " + t, args...)
}

var ErrDone		= errorf("done")
var ErrStopped		= errorf("stopped")

////////////////////////////////////////////////////////////////////////////////

type Script interface {
	Bind(Scriptable) (Releasable, error)
	Execute(context.Context, string, ...interface{}) error
	Release()
}

type Releasable interface {
	Release()
}

type Scriptable interface {
	Methods() []string
	Control(string, ...Value) (interface{}, error)
}

////////////////////////////////////////////////////////////////////////////////

type Value interface {
	Int() int
	String() string
}

type value struct {
	val interface{}
}

func (v value) Int() int {
	switch t := v.val.(type) {
	case int:
		return t
	case string:
		n, _ := strconv.ParseInt(t, 10, 32)
		return int(n)
	default:
		panic(1)
	}
}

func (v value) String() string {
	switch t := v.val.(type) {
	case int:
		return fmt.Sprintf("%d", t)
	case string:
		return t
	default:
		panic(1)
	}
}

func NewValue(v interface{}) Value {
	switch v.(type) {
	case int, string:
		return value{ val: v }
	default:
		panic(errorf("value type is not supported"))
	}
}

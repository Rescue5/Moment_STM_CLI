package lua

import (
	"github.com/yuin/gopher-lua"

	dms "dronmotors/dmetrics/internal/script"
)

type value struct {
	val lua.LValue
}

func (v value) Int() int {
	return int(lua.LVAsNumber(v.val))
}

func (v value) String() string {
	return string(lua.LVAsString(v.val))
}

func NewValue(v interface{}) dms.Value {
	switch n := v.(type) {
	case int:
		return value{ lua.LNumber(n) }
	case string:
		return value{ lua.LString(n) }
	default:
		panic(errorf("value type is not supported"))
	}
}

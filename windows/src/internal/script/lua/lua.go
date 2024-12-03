package lua

import (
	"fmt"

	"sync"
	"time"
	"strconv"
	"strings"
	"context"

	"layeh.com/gopher-luar"
	"github.com/yuin/gopher-lua"
	"github.com/yuin/gluamapper"

	dms "dronmotors/dmetrics/internal/script"
)

func errorf(t string, args ...interface{}) error {
	return fmt.Errorf("script/lua: " + t, args...)
}

type sfnstbl struct {
	Test *lua.LFunction
	OnConnect *lua.LFunction
	OnTelemetry *lua.LFunction
	OnDisconnect *lua.LFunction
}

type script struct {
	sync.Mutex // threads lock

	l *lua.LState
	fns sfnstbl
	params map[string]string

	threads map[string]*thread
}

type thread struct {
	l *lua.LState
	fun *lua.LFunction
	cancel context.CancelFunc
}

func (s *script) newScript(text string) error {
	s.l.SetContext(context.Background())
	if err := s.l.DoString(text); err != nil {
		return err
	} else if res, ok := s.l.Get(-1).(*lua.LTable); !ok {
		return errorf("no table found")
	} else if err := gluamapper.Map(res, &s.fns); err != nil {
		return err
	} else if s.fns.Test == nil {
		return errorf("no test function found")
	} else {
		return nil
	}
}

type releasable struct {
	release func()
}

func (r *releasable) Release() {
	r.release()
}

func (s *script) setGlobals(provider dms.Scriptable) {
	s.l.Register("stop", func(L *lua.LState) int {
		if msg := L.ToString(1); len(msg) > 0 {
			fmt.Println(errorf(msg))
		}
		panic(dms.ErrStopped)
		return 0
	})

	s.l.Register("sleep", func(L *lua.LState) int {
		select {
		case <-L.Context().Done():
		case <-time.After(time.Duration(L.ToInt(1)) * time.Millisecond):
		}
		return 0
	})

	s.l.Register("intParam", func(L *lua.LState) int {
		arg := L.ToString(1)

		if v, ok := s.params[arg]; ok {
			if n, err := strconv.Atoi(v); err != nil {
				L.Push(lua.LNumber(L.ToInt(2)))
			} else {
				L.Push(lua.LNumber(n))
			}
		} else {
			L.Push(lua.LNumber(L.ToInt(2)))
		}

		return 1
	})
	
	s.l.Register("strParam", func(L *lua.LState) int {
		arg := L.ToString(1)

		if v, ok := s.params[arg]; ok {
			L.Push(lua.LString(v))
		} else {
			L.Push(lua.LString(L.ToString(2)))
		}

		return 1
	})

}

func (s *script) releaseGlobals() {
	s.l.SetGlobal("stop", nil)
	s.l.SetGlobal("sleep", nil)
	s.l.SetGlobal("intParam", nil)
	s.l.SetGlobal("strParam", nil)
}

func (s *script) Bind(provider dms.Scriptable) (dms.Releasable, error) {
	s.Lock()
	defer s.Unlock()
	s.setGlobals(provider)

	for _, m := range provider.Methods() {
		method := m
		s.l.Register(method, func(L *lua.LState) int {
			args := []dms.Value{
				value{ L.Get(1) },
				value{ L.Get(2) },
				// FIXME: add more values, if needed
			}

			if v, err := provider.Control(method, args...); err != nil {
				panic(err)
			} else if (v != nil) {
				switch n := v.(type) {
				case int:
					L.Push(lua.LNumber(n))
				case string:
					L.Push(lua.LString(n))
				default:
					fmt.Println(errorf("control return value type is not supported"))
					return 0
				}

				return 1
			} else {
				return 0 // nil case
			}
		})
	}

	return &releasable{
		release: func() {
			s.Lock()
			defer s.Unlock()
			for _, method := range provider.Methods() {
				s.l.SetGlobal(method, nil)
			}
			s.releaseGlobals()
		},
	}, nil
}

func (s *script) getfunc(name string) *lua.LFunction {
	switch strings.ToLower(name) {
	case "test":
		return s.fns.Test
	case "onconnect":
		return s.fns.OnConnect
	case "ontelemetry":
		return s.fns.OnTelemetry
	case "ondisconnect":
		return s.fns.OnDisconnect
	}

	if v := s.l.GetGlobal(name); v.Type() == lua.LTFunction {
		return v.(*lua.LFunction)
	} else {
		return nil
	}
}

func (s *script) Execute(ctx context.Context, entry string, args ...interface{}) error {
	var largs []lua.LValue

	for i := range(args) {
		if v := luar.New(s.l, args[i]); v != lua.LNil {
			largs = append(largs, v)
		}
	}

	s.Lock()

	t, ok := s.threads[entry]
	if !ok {
		v := s.getfunc(entry)
		if v == nil {
			s.Unlock() // no such function => bail out
			return nil
		}

		l, c := s.l.NewThread()
		l.SetContext(ctx)

		t = &thread{
			l: l,
			fun: v,
			cancel: c,
		}

		s.threads[entry] = t
	}

	s.Unlock()

	if err := t.l.CallByParam(lua.P{ Fn: t.fun, Protect: true }, largs...); err != nil {
		if strings.Contains(err.Error(), "in function 'stop'") {
			return dms.ErrStopped
		} else {
			return err
		}
	}

	return nil
}

func (s *script) Release() {
	s.l.Close()
}

func NewScript(text string, params map[string]string) (dms.Script, error) {
	s := &script{
		params: params,
		l: lua.NewState(),
		threads: make(map[string]*thread),
	}

	if err := s.newScript(text); err != nil {
		return nil, err
	} else {
		return s, nil
	}
}

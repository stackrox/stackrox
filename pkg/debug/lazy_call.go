package debug

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

var (
	errEmpty = errors.New("empty LazyCall")
)

type lazyCall []interface{}

// LazyCall provides a way to call a function lazily. This is useful for debug logs, where you don't want a potentially
// expensive function evaluated if the respective log level is disabled.
func LazyCall(args ...interface{}) fmt.Stringer {
	return lazyCall(args)
}

func (c lazyCall) call() (vals []reflect.Value, err error) {
	if len(c) == 0 {
		return nil, errEmpty
	}
	fn := reflect.ValueOf(c[0])
	fnTy := fn.Type()
	if fnTy.Kind() != reflect.Func {
		return nil, fmt.Errorf("first LazyCall arg not a function, but %v", fnTy.Kind())
	}

	minArgs := fnTy.NumIn()
	maxArgs := fnTy.NumIn()
	if fnTy.IsVariadic() {
		minArgs--
		maxArgs = -1
	}

	argVals := make([]reflect.Value, 0, len(c)-1)
	for i, arg := range c[1:] {
		if lc, _ := arg.(lazyCall); lc != nil {
			rets, err := lc.call()
			if err != nil {
				return nil, errors.Wrapf(err, "evaluating argument %d", i+1)
			}
			argVals = append(argVals, rets...)
		} else {
			argVals = append(argVals, reflect.ValueOf(arg))
		}
	}

	if len(argVals) < minArgs {
		return nil, fmt.Errorf("invalid number of arguments: need at least %d, got %d", minArgs, len(argVals))
	}
	if maxArgs != -1 && len(argVals) > maxArgs {
		return nil, fmt.Errorf("invalid number of arguments: need at most %d, got %d", maxArgs, len(argVals))
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PANIC executing lazy call: %v", r)
		}
	}()

	vals = fn.Call(argVals)
	return
}

func (c lazyCall) String() string {
	vals, err := c.call()
	if err != nil {
		return fmt.Sprintf("<!%v>", err)
	}

	strs := make([]string, 0, len(vals))
	for _, val := range vals {
		strs = append(strs, fmt.Sprint(val.Interface()))
	}
	return strings.Join(strs, ", ")
}

package eval

import (
	"dghaehre/raja/lib"
	_ "embed"
	"fmt"
	"strings"
)

func (c *Context) LoadLib(name string) (Value, *runtimeError) {
	program, ok := lib.Stdlibs[name]
	if !ok {
		return nil, &runtimeError{
			reason: fmt.Sprintf("%s is not a valid standard library; could not import", name),
		}
	}

	v, err := c.Eval(strings.NewReader(program), "")
	if err != nil {
		if runtimeErr, ok := err.(*runtimeError); ok {
			return nil, runtimeErr
		} else {
			return nil, &runtimeError{
				reason: fmt.Sprintf("Error loading %s: %s", name, err.Error()),
			}
		}
	}
	return v, nil
}

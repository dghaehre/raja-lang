package lib

import (
	_ "embed"
)

//go:embed base.raja
var libBase string

var Stdlibs = map[string]string{
	"base": libBase,
}

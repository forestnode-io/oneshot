package commands

import (
	"errors"
	"strings"
)

type OutputFlagArg struct {
	Format string
	Opts   []string
}

func (o *OutputFlagArg) String() string {
	s := o.Format
	if 0 < len(o.Opts) {
		s += "=" + strings.Join(o.Opts, ",")
	}
	return s
}

func (o *OutputFlagArg) Set(v string) error {
	switch {
	case strings.HasPrefix(v, "json"):
		o.Format = "json"
		parts := strings.Split(v, "=")
		if len(parts) < 2 {
			return nil
		}
		o.Opts = strings.Split(parts[1], ",")
		return nil
	}
	return errors.New(`must be "json[=opts...]"`)
}

func (o *OutputFlagArg) Type() string {
	return "string"
}

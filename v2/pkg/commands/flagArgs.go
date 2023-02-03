package commands

import (
	"errors"
	"regexp"
	"strconv"
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

var sizeRe = regexp.MustCompile(`([1-9]\d*)([kmgtKMGT]?[i]?[bB])`)

func ParseSize(s string) (int64, error) {
	const (
		k  = 1000
		ki = 1024
	)

	parts := sizeRe.FindStringSubmatch(s)
	if len(parts) != 3 {
		return 0, errors.New("invalid size")
	}
	ns := parts[1]
	units := parts[2]

	n, err := strconv.ParseInt(ns, 10, 64)
	if err != nil {
		return 0, err
	}

	var (
		mult      int64 = 1
		usingBits       = false
	)
	switch len(units) {
	case 1:
		if units[0] == 'b' {
			usingBits = true
		}
	case 2:
		if units[1] == 'b' {
			usingBits = true
		}

		order := strings.ToLower(string(units[0]))
		switch order {
		case "k":
			mult = k
		case "m":
			mult = k * k
		case "g":
			mult = k * k * k
		case "t":
			mult = k * k * k * k
		}
	case 3:
		if units[2] == 'b' {
			usingBits = true
		}

		order := strings.ToLower(string(units[0]))
		switch order {
		case "k":
			mult = ki
		case "m":
			mult = ki * ki
		case "g":
			mult = ki * ki * ki
		case "t":
			mult = ki * ki * ki * ki
		}
	}

	if usingBits {
		if 1 < mult {
			mult /= 8
		} else {
			bumpByOne := n%8 != 0
			n /= 8
			if bumpByOne {
				n += 1
			}
		}
	}

	return mult * n, nil
}

type SizeFlagArg int

func (s *SizeFlagArg) String() string {
	return strconv.Itoa(int(*s)) + "B"
}

func (s *SizeFlagArg) Set(v string) error {
	size, err := ParseSize(v)
	if err != nil {
		return err
	}
	*s = SizeFlagArg(size)
	return nil
}

func (s *SizeFlagArg) Type() string {
	return "string"
}

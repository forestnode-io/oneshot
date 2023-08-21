package flagargs

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ArchiveMethod string

func (a *ArchiveMethod) String() string {
	return string(*a)
}

func (a *ArchiveMethod) Set(value string) error {
	switch value {
	case "zip", "tar", "tar.gz":
		*a = ArchiveMethod(value)
		return nil
	default:
		return fmt.Errorf(`invalid archive method %q, must be "zip", "tar" or "tar.gz`, value)
	}
}

func (a ArchiveMethod) Type() string {
	return "string"
}

type OutputFormat struct {
	Format string   `mapstructure:"format" yaml:"format"`
	Opts   []string `mapstructure:"opts" yaml:"opts"`
}

func (o *OutputFormat) String() string {
	s := o.Format
	if 0 < len(o.Opts) {
		s += "=" + strings.Join(o.Opts, ",")
	}
	return s
}

func (o *OutputFormat) Set(v string) error {
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

func (o *OutputFormat) Type() string {
	return "string"
}

type Size int

func (s *Size) String() string {
	return strconv.Itoa(int(*s)) + "B"
}

func (s *Size) Set(v string) error {
	size, err := parseSize(v)
	if err != nil {
		return err
	}
	*s = Size(size)
	return nil
}

func (s *Size) Type() string {
	return "string"
}

var sizeRe = regexp.MustCompile(`([1-9]\d*)([kmgtKMGT]?[i]?[bB])`)

func parseSize(s string) (int64, error) {
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

type HTTPHeader []string

func (h *HTTPHeader) SetValue(key, value string) {
	escapedKey := strings.ReplaceAll(key, ",", "\\,")
	escapedKey = strings.ReplaceAll(escapedKey, "=", "\\=")
	escapedValue := strings.ReplaceAll(value, ",", "\\,")
	escapedValue = strings.ReplaceAll(escapedValue, "=", "\\=")

	*h = append(*h, fmt.Sprintf("%s=%s", escapedKey, escapedValue))
}

func (h *HTTPHeader) GetValue(key string) ([]string, bool) {
	var values []string
	found := false

	for _, pair := range *h {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		decodedKey := strings.ReplaceAll(kv[0], "\\,", ",")
		decodedKey = strings.ReplaceAll(decodedKey, "\\=", "=")

		if decodedKey == key {
			decodedValue := strings.ReplaceAll(kv[1], "\\,", ",")
			decodedValue = strings.ReplaceAll(decodedValue, "\\=", "=")
			values = append(values, decodedValue)
			found = true
		}
	}

	return values, found
}

func (h *HTTPHeader) Inflate() map[string][]string {
	return unflatten(*h)
}

func (h *HTTPHeader) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var header map[string][]string
	if err := unmarshal(&header); err != nil {
		return err
	}

	*h = flatten(header)
	return nil
}

func (h *HTTPHeader) MarshalYAML() (interface{}, error) {
	return unflatten(*h), nil
}

func flatten(header map[string][]string) []string {
	var flattened []string

	for key, values := range header {
		for _, value := range values {
			escapedKey := strings.ReplaceAll(key, ",", "\\,")
			escapedKey = strings.ReplaceAll(escapedKey, "=", "\\=")
			escapedValue := strings.ReplaceAll(value, ",", "\\,")
			escapedValue = strings.ReplaceAll(escapedValue, "=", "\\=")

			flattened = append(flattened, fmt.Sprintf("%s=%s", escapedKey, escapedValue))
		}
	}

	return flattened
}

func unflatten(flattened []string) map[string][]string {
	header := make(map[string][]string)

	for _, pair := range flattened {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.ReplaceAll(kv[0], "\\,", ",")
		key = strings.ReplaceAll(key, "\\=", "=")
		value := strings.ReplaceAll(kv[1], "\\,", ",")
		value = strings.ReplaceAll(value, "\\=", "=")

		header[key] = append(header[key], value)
	}

	return header
}

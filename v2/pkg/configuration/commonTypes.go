package configuration

import (
	"errors"
	"os"
)

type SingletonOrSlice[T any] []T

type PathOrContent struct {
	Path    string `mapstructure:"path" yaml:"path"`
	Content string `mapstructure:"content" yaml:"content"`
}

func (poc *PathOrContent) Validate() error {
	if poc.Path != "" && poc.Content != "" {
		return errors.New("only one of path or content can be specified")
	}
	return nil
}

func (poc *PathOrContent) GetContent() ([]byte, error) {
	if poc.Content != "" {
		return []byte(poc.Content), nil
	}
	if poc.Path == "" {
		return nil, errors.New("no content or path specified")
	}

	return os.ReadFile(poc.Path)
}

func (poc *PathOrContent) IsZero() bool {
	if poc == nil {
		return true
	}
	return poc.Path == "" && poc.Content == ""
}

type FileExport struct {
	Path string `mapstructure:"path" yaml:"path"`
	Mode string `mapstructure:"mode" yaml:"mode"`
}

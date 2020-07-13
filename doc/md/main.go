package main

import (
	"bytes"
	"github.com/raphaelreyna/oneshot/cmd"
	"github.com/spf13/cobra/doc"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	cmd.SetFlags()
	mdBuffer := &bytes.Buffer{}
	err := doc.GenMarkdown(cmd.RootCmd, mdBuffer)
	if err != nil {
		panic(err)
	}

	parts := bytes.Split(mdBuffer.Bytes(), []byte(`### Synopsis`))
	os.Stdout.Write(parts[0])

	here, err := os.Open(".")
	defer here.Close()
	if err != nil {
		panic(err)
	}
	files, _ := here.Readdirnames(0)
	sort.Strings(files)
	var file *os.File
	for _, fName := range files {
		if filepath.Ext(fName) == ".md" {
			file, err = os.Open(fName)
			if err != nil {
				panic(err)
			}
			os.Stdout.Write([]byte("\n"))
			io.Copy(os.Stdout, file)
			file.Close()
			os.Stdout.Write([]byte("\n"))
		}
	}

	os.Stdout.Write([]byte("\n### Synopsis\n"))
	os.Stdout.Write(parts[1])
}

package main

import (
	"bytes"
	"github.com/forestnode-io/oneshot/cmd"
	"github.com/spf13/cobra/doc"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
)

const logo = `<img src="https://github.com/forestnode-io/oneshot/raw/master/oneshot_banner.png" width="744px" height="384px">

`

func main() {
	app, err := cmd.NewApp()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	app.SetFlags()

	mdBuffer := &bytes.Buffer{}
	err = doc.GenMarkdown(app.Cmd(), mdBuffer)
	if err != nil {
		panic(err)
	}

	// Logo
	os.Stdout.Write([]byte(logo))

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

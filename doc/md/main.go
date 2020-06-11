package main

import (
	"github.com/raphaelreyna/oneshot/cmd"
	"github.com/spf13/cobra/doc"
	"log"
	"os"
)

func main() {
	cmd.SetFlags()
	err := doc.GenMarkdown(cmd.RootCmd, os.Stdout)
	if err != nil {
		log.Print(err)
	}
}

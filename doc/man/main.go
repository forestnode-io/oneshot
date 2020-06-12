package main

import (
	"github.com/raphaelreyna/oneshot/cmd"
	"github.com/spf13/cobra/doc"
	"log"
	"os"
)

func main() {
	cmd.SetFlags()
	header := doc.GenManHeader{
		Title:   "ONESHOT",
		Section: "1",
		Source:  "https://github.com/raphaelreyna/oneshot",
	}
	err := doc.GenMan(cmd.RootCmd, &header, os.Stdout)
	if err != nil {
		log.Print(err)
	}
}

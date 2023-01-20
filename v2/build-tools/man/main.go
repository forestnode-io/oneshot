package main

import (
	"log"
	"os"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands/root"
	"github.com/spf13/cobra/doc"
)

func main() {
	cmd := root.CobraCommand()
	header := doc.GenManHeader{
		Title:   "ONESHOT",
		Section: "1",
		Source:  "https://github.com/raphaelreyna/oneshot/v2",
	}
	if err := doc.GenMan(cmd, &header, os.Stdout); err != nil {
		log.Print(err)
	}
}

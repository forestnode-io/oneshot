package main

import (
	"log"
	"os"

	"github.com/forestnode-io/oneshot/v2/pkg/commands/root"
	"github.com/spf13/cobra/doc"
)

func main() {
	cmd := root.CobraCommand(false)
	header := doc.GenManHeader{
		Title:   "ONESHOT",
		Section: "1",
		Source:  "https://github.com/forestnode-io/oneshot/v2",
	}
	if err := doc.GenManTree(cmd, &header, os.Args[1]); err != nil {
		log.Print(err)
	}
}

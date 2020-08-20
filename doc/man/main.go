package main

import (
	"github.com/raphaelreyna/oneshot/cmd"
	"github.com/spf13/cobra/doc"
	"log"
	"os"
)

func main() {
	app, err := cmd.NewApp()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	app.SetFlags()
	header := doc.GenManHeader{
		Title:   "ONESHOT",
		Section: "1",
		Source:  "https://github.com/raphaelreyna/oneshot",
	}
	err = doc.GenMan(app.Cmd(), &header, os.Stdout)
	if err != nil {
		log.Print(err)
	}
}

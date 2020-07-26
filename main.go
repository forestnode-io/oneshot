package main

import (
	"github.com/raphaelreyna/oneshot/cmd"
	"log"
	"os"
)

func main() {
	app, err := cmd.NewApp()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	app.Start()
}

package main

import (
	"github.com/forestnode-io/oneshot/cmd"
	"log"
	"math/rand"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	app, err := cmd.NewApp()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	app.Start()
}

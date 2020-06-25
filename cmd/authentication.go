package cmd

import (
	"bufio"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
)

func setupUsernamePassword() error {
	if passwordHidden {
		os.Stdout.WriteString("password: ")
		passreader := bufio.NewReader(os.Stdin)
		passwordBytes, err := passreader.ReadString('\n')
		if err != nil {
			return err
		}
		password = string(passwordBytes)
		password = strings.TrimSpace(password)
		os.Stdout.WriteString("\n")
	} else if passwordFile != "" {
		passwordBytes, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			return err
		}
		password = string(passwordBytes)
		password = strings.TrimSpace(password)
	}
	return nil
}

func randomPassword() string {
	const lowerChars = "abcdefghijklmnopqrstuvwxyz"
	const upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const numericChars = "1234567890"

	var defSeperator = "-"

	runes := []rune(lowerChars + upperChars + numericChars)
	l := len(runes)
	password := ""
	for i := 1; i < 15; i++ {
		if i%5 == 0 {
			password += defSeperator
			continue
		}
		password += string(runes[rand.Intn(l)])
	}
	return password
}

func randomUsername() string {
	adjs := [...]string{"bulky", "fake", "artistic", "plush", "ornate", "kind", "nutty", "miniature", "huge", "evergreen", "several", "writhing", "scary", "equatorial", "obvious", "rich", "beneficial", "actual", "comfortable", "well-lit"}

	nouns := [...]string{"representative", "prompt", "respond", "safety", "blood", "fault", "lady", "routine", "position", "friend", "uncle", "savings", "ambition", "advice", "responsibility", "consist", "nobody", "film", "attitude", "heart"}

	l := len(adjs)

	return adjs[rand.Intn(l)] + "_" + nouns[rand.Intn(l)]
}

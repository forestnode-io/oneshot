package conf

import (
	"bufio"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
)

func (c *Conf) SetupCredentials() error {
	if c.PasswordHidden {
		// Read password from standard in
		os.Stdout.WriteString("password: ")
		passreader := bufio.NewReader(os.Stdin)
		passwordBytes, err := passreader.ReadString('\n')
		if err != nil {
			return err
		}
		c.Password = string(passwordBytes)
		c.Password = strings.TrimSpace(c.Password)
		os.Stdout.WriteString("\n")
	} else if c.PasswordFile != "" {
		// Read password from file
		passwordBytes, err := ioutil.ReadFile(c.PasswordFile)
		if err != nil {
			return err
		}
		c.Password = string(passwordBytes)
		c.Password = strings.TrimSpace(c.Password)
	}

	if c.cmdFlagSet.Changed("username") && c.Username == "" {
		c.Username = randomUsername()
		c.randUser = true
	}
	if c.cmdFlagSet.Changed("password") && c.Password == "" {
		c.Password = randomPassword()
		c.randPass = true
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

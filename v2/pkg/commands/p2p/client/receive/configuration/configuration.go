package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	OfferFile  string `json:"offerFile" yaml:"offerFile"`
	AnswerFile string `json:"answerFile" yaml:"answerFile"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("receive flags", pflag.ExitOnError)

	c.fs.String("offer-file", "", "Path to the file containing the offer.")
	c.fs.String("answer-file", "", "Path to the file containing the answer.")

	cobra.AddTemplateFunc("receiveFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("offer-file") {
		c.OfferFile, _ = c.fs.GetString("offer-file")
	}
	if c.fs.Changed("answer-file") {
		c.AnswerFile, _ = c.fs.GetString("answer-file")
	}
}

func (c *Configuration) Validate() error {
	return nil
}

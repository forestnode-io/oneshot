package main

import (
	"bytes"
	"encoding/json"
	"testing"

	itest "github.com/forestnode-io/oneshot/v2/integration_testing"
	"github.com/stretchr/testify/suite"
)

func TestBasicTestSuite(t *testing.T) {
	suite.Run(t, new(ts))
}

type ts struct {
	itest.TestSuite
}

func (suite *ts) Test_NoHang() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"version"}
	oneshot.Start()
	defer oneshot.Cleanup()

	oneshot.Wait()
}

func (suite *ts) Test_NoError() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"version"}
	oneshot.Start()
	defer oneshot.Cleanup()

	oneshot.Wait()
	suite.Assert().True(oneshot.Cmd.ProcessState.Success())
}

func (suite *ts) Test_JSON() {
	var (
		oneshot = suite.NewOneshot()
		stdout  = bytes.NewBuffer(nil)
	)
	oneshot.Args = []string{"version", "--output=json"}
	oneshot.Stdout = stdout
	oneshot.Start()
	defer oneshot.Cleanup()

	oneshot.Wait()
	suite.Assert().True(oneshot.Cmd.ProcessState.Success())

	var output struct {
		APIVersion string `json:"apiVersion"`
		Version    string `json:"version"`
		License    string `json:"license"`
	}

	err := json.NewDecoder(stdout).Decode(&output)
	suite.Require().NoError(err)

	suite.Assert().NotEmpty(output.APIVersion)
	suite.Assert().NotEmpty(output.Version)
	suite.Assert().NotEmpty(output.License)
}

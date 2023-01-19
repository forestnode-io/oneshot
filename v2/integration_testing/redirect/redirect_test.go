package main

import (
	"bytes"
	"io"
	"net/http"

	itest "github.com/raphaelreyna/oneshot/v2/integration_testing"
)

func (suite *ts) Test_StdinTTY_StderrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"redirect", "https://github.com"}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDOUT=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusTemporaryRedirect, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	suite.Assert().Contains(string(body), "https://github.com")

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stdout))

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stderr), "")
}

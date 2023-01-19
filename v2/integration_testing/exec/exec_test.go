package main

import (
	"bytes"
	"io"
	"net/http"
	"runtime"
	"strings"

	itest "github.com/raphaelreyna/oneshot/v2/integration_testing"
)

func (suite *ts) Test_StdinTTY_StderrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"exec", "go", "env", "GOOS"}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDIN=true",
		"ONESHOT_TESTING_TTY_STDOUT=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()

	output := strings.ReplaceAll(string(body), "\n", "")
	suite.Assert().Equal(runtime.GOOS, output)

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stdout))

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stderr))
}

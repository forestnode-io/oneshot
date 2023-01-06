package main

import (
	"bytes"
	"io"
	"net/http"
	"runtime"
	"strings"
)

func (suite *BasicTestSuite) Test_Exec() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"exec", "go", "env", "GOOS"}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := retryClient{}
	resp, err := client.get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()

	output := strings.ReplaceAll(string(body), "\n", "")
	suite.Assert().Equal(runtime.GOOS, output)

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "\x1b[?25h")
}

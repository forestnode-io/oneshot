package main

import (
	"bytes"
	"io"
	"net/http"
)

func (suite *BasicTestSuite) Test_Redirect() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"redirect", "https://github.com"}
	oneshot.Start()

	client := retryClient{}
	resp, err := client.get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusTemporaryRedirect, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	suite.Assert().Contains(string(body), "https://github.com")

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "\x1b[?25h")
}

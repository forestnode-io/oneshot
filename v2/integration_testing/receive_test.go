package main

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
)

func (suite *BasicTestSuite) Test_Receive_ToStdout() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive"}
	oneshot.Start()
	defer oneshot.Cleanup()

	client := retryClient{}
	resp := client.post("http://127.0.0.1:8080", "text/plain", bytes.NewReader([]byte("SUCCESS")))
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("SUCCESS", string(stdout))
}

func (suite *BasicTestSuite) Test_Receive_ToFile() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive", ".", "--name=test.txt"}
	oneshot.Start()
	defer oneshot.Cleanup()

	client := retryClient{}
	resp := client.post("http://127.0.0.1:8080", "text/plain", bytes.NewReader([]byte("SUCCESS")))
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	oneshot.Wait()
	fileContents, err := os.ReadFile(filepath.Join(oneshot.WorkingDir, "test.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS", string(fileContents))
}

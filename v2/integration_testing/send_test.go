package main

import (
	"bytes"
	"io"
	"net/http"
)

func (suite *BasicTestSuite) Test_Send_FromStdin() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "-p", "8081"}
	oneshot.Stdin = EOFReader([]byte("SUCCESS"))
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := retryClient{}
	resp, err := client.get("http://127.0.0.1:8081")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "transfer complete")
}

func (suite *BasicTestSuite) Test_Send_File() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "./test.txt"}
	oneshot.Files = FilesMap{"./test.txt": []byte("SUCCESS")}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := retryClient{}
	resp, err := client.get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "transfer complete")
}

func (suite *BasicTestSuite) Test_Send_StatusCode() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "--status-code", "418"}
	oneshot.Stdin = EOFReader([]byte("SUCCESS"))
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := retryClient{}
	resp, err := client.get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusTeapot)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "transfer complete")
}

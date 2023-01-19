package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"

	itest "github.com/raphaelreyna/oneshot/v2/integration_testing"
)

func (suite *ts) Test_FROM_ANY_TO_StdoutTTY__StderrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive"}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDOUT=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	client := itest.RetryClient{
		Suite: &suite.Suite,
	}
	resp, err := client.Post("http://127.0.0.1:8080", "text/plain", bytes.NewReader([]byte("SUCCESS")))
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("SUCCESS\n", string(stdout))

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stderr))
}

func (suite *ts) Test_FROM_ANY_TO_StdoutTTY__StderrNONTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive"}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDOUT=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	client := itest.RetryClient{
		Suite: &suite.Suite,
	}
	resp, err := client.Post("http://127.0.0.1:8080", "text/plain", bytes.NewReader([]byte("SUCCESS")))
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("SUCCESS\n", string(stdout))

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stderr), "success\n")
}

func (suite *ts) Test_FROM_ANY_TO_File__StdoutTTY_StderrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive", "./test.txt"}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDOUT=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	client := itest.RetryClient{
		Suite: &suite.Suite,
	}
	resp, err := client.Post("http://127.0.0.1:8080", "text/plain", bytes.NewReader([]byte("SUCCESS")))
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	oneshot.Wait()
	fileContents, err := os.ReadFile(filepath.Join(oneshot.WorkingDir, "test.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS", string(fileContents))

	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "success\n\x1b[?25h")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stderr), "")
}

func (suite *ts) Test_FROM_ANY_TO_StdoutTTY_DecodeBase64() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive", "--decode-b64"}
	oneshot.Start()
	defer oneshot.Cleanup()

	var (
		payload        = []byte("SUCCESS")
		encodedPayload = make([]byte, base64.StdEncoding.EncodedLen(len(payload)))
	)
	base64.StdEncoding.Encode(encodedPayload, payload)
	client := itest.RetryClient{}
	resp, err := client.Post("http://127.0.0.1:8080", "text/plain", bytes.NewReader(encodedPayload))
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("SUCCESS\n", string(stdout))
}

func (suite *ts) Test_FROM_ANY_TO_File_DecodeBase64() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive", "./test.txt", "--decode-b64"}
	oneshot.Start()
	defer oneshot.Cleanup()

	var (
		payload        = []byte("SUCCESS")
		encodedPayload = make([]byte, base64.StdEncoding.EncodedLen(len(payload)))
	)
	base64.StdEncoding.Encode(encodedPayload, payload)
	client := itest.RetryClient{}
	resp, err := client.Post("http://127.0.0.1:8080", "text/plain", bytes.NewReader(encodedPayload))
	suite.Require().NoError(err)
	suite.Require().NotNil(resp)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	oneshot.Wait()
	fileContents, err := os.ReadFile(filepath.Join(oneshot.WorkingDir, "test.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS", string(fileContents))
}

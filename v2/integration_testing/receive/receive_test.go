package main

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

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

func (suite *ts) Test_MultipleClients() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive", "./test.txt"}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDOUT=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	m := sync.Mutex{}
	c := sync.NewCond(&m)

	responses := make(chan int, runtime.NumCPU())
	wg := sync.WaitGroup{}
	for i := 1; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func(index int) {
			c.L.Lock()
			c.Wait()
			c.L.Unlock()

			payload := []byte("SUCCESS")
			resp, _ := http.Post("http://127.0.0.1:8080", "text/plain", bytes.NewReader(payload))
			if resp != nil {
				if resp.Body != nil {
					resp.Body.Close()
				}
				responses <- resp.StatusCode
			} else {
				responses <- 0
			}
			wg.Done()
		}(i)
	}
	time.Sleep(500 * time.Millisecond)
	c.L.Lock()
	c.Broadcast()
	c.L.Unlock()

	wg.Wait()
	close(responses)

	oks := 0
	for code := range responses {
		if code == 200 {
			oks++
		}
	}
	suite.Assert().Equal(1, oks)

	oneshot.Wait()
	fileContents, err := os.ReadFile(filepath.Join(oneshot.WorkingDir, "test.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS", string(fileContents))

	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "success\n\x1b[?25h")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stderr), "")
}

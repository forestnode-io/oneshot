package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	itest "github.com/raphaelreyna/oneshot/v2/integration_testing"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
)

func (suite *ts) Test_FROM_StdinTTY_TO_ANY__StdoutTTY_StdoutErrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "-p", "8081"}
	oneshot.Stdin = itest.EOFReader([]byte("SUCCESS"))
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDIN=true",
		"ONESHOT_TESTING_TTY_STDOUT=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8081")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()

	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("\n", string(stdout))

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stderr))
}

func (suite *ts) Test_FROM_StdinTTY_TO_ANY__StdoutNONTTY_StderrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "-p", "8081"}
	oneshot.Stdin = itest.EOFReader([]byte("SUCCESS"))
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDIN=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8081")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "success\n")
	suite.Assert().NotContains(string(stdout), "\x1b")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stderr))
}

func (suite *ts) Test_FROM_File_TO_ANY__StdoutTTY_StderrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "./test.txt"}
	oneshot.Files = itest.FilesMap{"./test.txt": []byte("SUCCESS")}
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
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "success\n\x1b[?25h")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stderr))
}

func (suite *ts) Test_FROM_File_TO_ANY__StdoutNONTTY_StderrTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "./test.txt"}
	oneshot.Files = itest.FilesMap{"./test.txt": []byte("SUCCESS")}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDIN=true",
		"ONESHOT_TESTING_TTY_STDERR=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	// expect dynamic output to have gone to stderr but static output goes to stdout
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "success\n")
	suite.Assert().NotContains(string(stdout), "\x1b")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stderr), "success\n\x1b[?25h")
}

func (suite *ts) Test_FROM_File_TO_ANY__StdoutNONTTY_StderrNONTTY() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "./test.txt"}
	oneshot.Files = itest.FilesMap{"./test.txt": []byte("SUCCESS")}
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDIN=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	// expect no dynamic out, only static outpu ton stdout
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "success\n")
	suite.Assert().NotContains(string(stdout), "\x1b")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stderr))
}

func (suite *ts) Test_FROM_File_TO_ANY__JSON() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "--output", "json", "./test.txt"}
	oneshot.Files = itest.FilesMap{"./test.txt": []byte("SUCCESS")}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	suite.Assert().NoError(err)
	resp.Body.Close()
	suite.Assert().Equal(string(body), "SUCCESS")

	oneshot.Wait()
	// expect no dynamic out, only static outpu ton stdout
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	var report output.Report
	err = json.Unmarshal(stdout, &report)
	suite.Assert().NoError(err)

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Equal("", string(stderr))
}

func (suite *ts) Test_StatusCode() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "--status-code", "418"}
	oneshot.Stdin = itest.EOFReader([]byte("SUCCESS"))
	oneshot.Env = []string{
		"ONESHOT_TESTING_TTY_STDIN=true",
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(resp.StatusCode, http.StatusTeapot)
}

func (suite *ts) Test_Send_Directory() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "./testDir"}
	oneshot.Files = itest.FilesMap{
		"./testDir/test.txt":  []byte("SUCCESS"),
		"./testDir/test2.txt": []byte("SUCCESS2"),
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := itest.RetryClient{}
	resp, err := client.Get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)

	tarFileName := filepath.Join(suite.TestDir, "test.tar")
	bufBytes, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	err = os.WriteFile(tarFileName, bufBytes, 0600)
	suite.Require().NoError(err)

	tarOut, err := exec.Command("tar", "-xf", tarFileName, "-C", suite.TestDir).CombinedOutput()
	suite.Require().NoError(err, string(tarOut))

	fileBytes, err := os.ReadFile(filepath.Join(suite.TestDir, "testDir", "test.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS", string(fileBytes))

	fileBytes, err = os.ReadFile(filepath.Join(suite.TestDir, "testDir", "test2.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS2", string(fileBytes))

	oneshot.Wait()
}

func (suite *ts) Test_MultipleClients() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send"}
	oneshot.Stdin = io.LimitReader(rand.Reader, 1<<15)
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

			resp, _ := http.Get("http://127.0.0.1:8080")
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
	gones := 0
	for code := range responses {
		if code == 200 {
			oks++
		} else if code == http.StatusGone {
			gones++
		}
	}
	suite.Assert().Equal(1, oks)
	suite.Assert().Equal(runtime.NumCPU()-2, gones)

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "success\n\x1b[?25h")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stderr), "")
}

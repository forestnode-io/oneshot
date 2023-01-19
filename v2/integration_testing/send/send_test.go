package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	itest "github.com/raphaelreyna/oneshot/v2/integration_testing"
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
	suite.Assert().Contains(string(stdout), "")

	stderr := oneshot.Stderr.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stderr), "")
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

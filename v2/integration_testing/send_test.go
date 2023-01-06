package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	suite.Assert().Contains(string(stdout), "...success\n\x1b[?25h")
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
	suite.Assert().Contains(string(stdout), "...success\n\x1b[?25h")
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
	suite.Assert().Contains(string(stdout), "...success\n\x1b[?25h")
}

func (suite *BasicTestSuite) Test_Send_Directory() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"send", "./testDir"}
	oneshot.Files = FilesMap{
		"./testDir/test.txt":  []byte("SUCCESS"),
		"./testDir/test2.txt": []byte("SUCCESS2"),
	}
	oneshot.Start()
	defer oneshot.Cleanup()

	// ---

	client := retryClient{}
	resp, err := client.get("http://127.0.0.1:8080")
	suite.Require().NoError(err)
	suite.Assert().Equal(http.StatusOK, resp.StatusCode)

	tarFileName := filepath.Join(suite.testDir, "test.tar")
	bufBytes, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	err = os.WriteFile(tarFileName, bufBytes, 0600)
	suite.Require().NoError(err)

	tarOut, err := exec.Command("tar", "-xf", tarFileName, "-C", suite.testDir).CombinedOutput()
	suite.Require().NoError(err, string(tarOut))

	fileBytes, err := os.ReadFile(filepath.Join(suite.testDir, "testDir", "test.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS", string(fileBytes))

	fileBytes, err = os.ReadFile(filepath.Join(suite.testDir, "testDir", "test2.txt"))
	suite.Require().NoError(err)
	suite.Assert().Equal("SUCCESS2", string(fileBytes))

	oneshot.Wait()
	stdout := oneshot.Stdout.(*bytes.Buffer).Bytes()
	suite.Assert().Contains(string(stdout), "...success\n\x1b[?25h")
}

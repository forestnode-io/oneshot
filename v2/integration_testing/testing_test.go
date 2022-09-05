package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestBasicTestSuite(t *testing.T) {
	suite.Run(t, new(BasicTestSuite))
}

type BasicTestSuite struct {
	suite.Suite
	testDir string
}

func (suite *BasicTestSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "")
	suite.Require().NoError(err)

	suite.Require().NoError(err)
	suite.testDir = tempDir

	cmdPath, err := filepath.EvalSymlinks("../build-output/oneshot")
	suite.Require().NoError(err)
	newCmdPath := filepath.Join(suite.testDir, "oneshot.testing")

	_, err = exec.Command("cp", cmdPath, newCmdPath).CombinedOutput()
	suite.Require().NoError(err)

	err = os.Chdir(suite.testDir)
	suite.Require().NoError(err)
}

func (suite *BasicTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.testDir)
	suite.Require().NoError(err)
}

func (suite *BasicTestSuite) NewOneshot() *Oneshot {
	dir, err := os.MkdirTemp(suite.testDir, "subtest*")
	suite.Require().NoError(err)
	return &Oneshot{
		T:          suite.T(),
		WorkingDir: dir,
	}
}

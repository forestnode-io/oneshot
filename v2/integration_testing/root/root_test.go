package main

import (
	"syscall"
	"testing"
	"time"

	itest "github.com/raphaelreyna/oneshot/v2/integration_testing"
	"github.com/stretchr/testify/suite"
)

func TestBasicTestSuite(t *testing.T) {
	suite.Run(t, new(ts))
}

type ts struct {
	itest.TestSuite
}

func (suite *ts) Test_Signal_SIGINT() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"receive"}
	oneshot.Start()
	defer oneshot.Cleanup()

	time.Sleep(500 * time.Millisecond)
	err := oneshot.Cmd.Process.Signal(syscall.SIGINT)
	suite.Require().NoError(err)

	oneshot.Wait()
}

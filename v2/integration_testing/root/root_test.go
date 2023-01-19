package main

import (
	"syscall"
	"time"
)

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

package main

import (
	"testing"

	itest "github.com/oneshot-uno/oneshot/v2/integration_testing"
	"github.com/stretchr/testify/suite"
)

func TestBasicTestSuite(t *testing.T) {
	suite.Run(t, new(ts))
}

type ts struct {
	itest.TestSuite
}

func (suite *ts) Test_NoHang() {
	var oneshot = suite.NewOneshot()
	oneshot.Args = []string{"version"}
	oneshot.Start()
	defer oneshot.Cleanup()

	oneshot.Wait()
}

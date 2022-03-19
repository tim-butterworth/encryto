package handshake_test

import (
	"fmt"
	"testing"
)

type TestHelper interface {
	ExpectStringsToMatch(actual string, expected string)
}

type testHelper struct {
	t *testing.T
}

func (helper *testHelper) ExpectStringsToMatch(actual string, expected string) {
	if actual != expected {
		helper.t.Log(fmt.Sprintf("Expected \n[%s]\n received \n[%s] ", expected, actual))
		helper.t.Fail()
	}
}

func newTestHelper(t *testing.T) TestHelper {
	return &testHelper{
		t: t,
	}
}

package testcat

import (
	"os"
	"testing"
)

type TestCat string

const (
	UnitTest    TestCat = "unit"
	Integration TestCat = "integration"
	Container   TestCat = "container"
	SmokeTest   TestCat = "smoke"
)

func (c TestCat) String() string {
	return string(c)
}

func CheckTestCategory(t *testing.T, c TestCat) {
	tCat := os.Getenv("CATEGORY")

	if tCat != c.String() {
		t.Skipf("set CATEGORY=%s to run this test", c.String())
	}
}

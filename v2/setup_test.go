package toolkit

import (
	"os"
	"testing"
)

var testTool Tools

func TestMain(m *testing.M) {
	exitCode := m.Run()

	os.Exit(exitCode)
}

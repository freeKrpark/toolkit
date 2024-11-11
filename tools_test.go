package toolkit

import "testing"

func TestTools_RandomString(t *testing.T) {
	var testTool Tools
	s := testTool.RandomString(10)

	if len(s) != 10 {
		t.Errorf("wrong length random String Returrend, expected 10, but got %d", len(s))
	}
}

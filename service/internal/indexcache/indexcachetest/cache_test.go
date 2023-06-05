package indexcachetest

import (
	"testing"
)

func Test_New(t *testing.T) {
	// Test that New doesn't panic and indexcache.Interface is implemented.
	New(Config{})
}

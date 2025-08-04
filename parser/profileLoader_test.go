package parser

import "testing"

func TestProfileLoader(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir() //ensure file does not exist

	defer FailTestIfPanic(t, t.Name())
	_ = loadProfiles(tempDir)
}

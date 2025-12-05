package parser

import (
	"testing"
)

func TestLoadGrades(t *testing.T) {

	_, err := loadGrades("../grade-data/")
	if err != nil {
		t.Errorf("loadGrades() error = %v", err)
	}

}

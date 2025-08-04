package parser

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProfileLoader(t *testing.T) {
	//remove side effects of the test
	clearGlobals()
	professors := make(map[string]*schema.Professor)
	professorIDMap := make(map[primitive.ObjectID]string)

	// profiles.json is just a copy of professors.json
	for _, prof := range testProfessors {
		professorKey := prof.First_name + prof.Last_name
		professors[professorKey] = &prof
		professorIDMap[prof.Id] = professorKey
	}

	log.SetOutput(&bytes.Buffer{})
	err := loadProfiles("testdata/coursebook/")
	log.SetOutput(os.Stdout)

	if err != nil {
		t.Errorf("faild with error %v", err)
	}

	if diff := cmp.Diff(professors, Professors); diff != "" {
		t.Errorf("profiles mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(professorIDMap, ProfessorIDMap); diff != "" {
		t.Errorf("profiles mismatch (-want +got):\n%s", diff)
	}
	clearGlobals()
}

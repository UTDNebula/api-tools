package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TestData struct {
	Input      string
	RowInfo    map[string]*goquery.Selection
	ClassInfo  map[string]string
	Section    schema.Section
	Course     schema.Course
	Professors []schema.Professor
}

var (
	// testData global map containing the data from /testdata/coursebook by folder name
	testData map[string]TestData

	// testCourses is the unmarshalled /testdata/coursebook/courses.json
	// created by running Parse on the /testdata/coursebook
	testCourses []schema.Course
	// testSections is the unmarshalled /testdata/coursebook/sections.json
	testSections []schema.Section
	// testProfessors is the unmarshalled /testdata/coursebook/professors.json
	testProfessors []schema.Professor
	// testProfiles is the unmarshalled /testdata/coursebook/profiles.json
	testProfiles []schema.Professor
)

// TestMain entry point for all tests in the parser package.
// This function will load all the test data in /testdata
//
//   - testData: Map of individual coursebook html files and there corresponding schema representations.
//     Used to test courseParser, sectionParser and professorParser
//
//   - testCourses, testSections and testProfessors: List of containing the schema respective representations of running
//     Parse on the /testdata/coursebook. User to test Parse and validator.
//
// You can optionally provide the flag `update`, which will run
// updateTestData. Example usage
//
// `go test -v ./parser -args -update`
func TestMain(m *testing.M) {
	update := flag.Bool("update", false, "Regenerates the expected output for the provided test inputs. Should only be used when you are 100% sure your code is correct! It will make all test pass :)")

	if !flag.Parsed() {
		flag.Parse()
	}

	if *update {
		log.Printf("Updating test data...")
		if err := updateTestData(); err != nil {
			log.Fatalf("Error updating test data: %v", err)
		}
		log.Println("Test data updated successfully!")
		os.Exit(0)
	}

	testData = make(map[string]TestData)
	dir, err := os.ReadDir("testdata/coursebook/")
	if err != nil {
		log.Fatalf("Failed to load testdata: %v", err)
	}

	for _, file := range dir {
		if !file.IsDir() {
			continue
		}
		filePath := filepath.Join("testdata/coursebook/", file.Name())

		if testData[file.Name()], err = loadTest(filePath); err != nil {
			log.Fatalf("Failed to load %s: %v", file.Name(), err)
		}
	}

	testCourses, err = utils.UnmarshallFile[[]schema.Course]("./testdata/coursebook/courses.json")
	if err != nil {
		log.Fatalf("Failed to load courses: %v", err)
	}
	testSections, err = utils.UnmarshallFile[[]schema.Section]("./testdata/coursebook/sections.json")
	if err != nil {
		log.Fatalf("Failed to load sections: %v", err)
	}
	testProfessors, err = utils.UnmarshallFile[[]schema.Professor]("./testdata/coursebook/professors.json")
	if err != nil {
		log.Fatalf("Failed to load professors: %v", err)
	}

	testProfiles, err = utils.UnmarshallFile[[]schema.Professor]("./testdata/coursebook/profiles.json")
	if err != nil {
		log.Fatalf("Failed to load profiles: %v", err)
	}

	os.Exit(m.Run())
}

// loadTest utility function for creating a TestData from a given directory.
// requires the passed directory to match the structure outlined in updateTestData
func loadTest(dir string) (result TestData, err error) {

	htmlBytes, err := os.ReadFile(filepath.Join(dir, "input.html"))
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return
	}
	result.Input = string(htmlBytes)
	result.RowInfo = getRowInfo(doc)
	result.Section, err = utils.UnmarshallFile[schema.Section](filepath.Join(dir, "section.json"))
	if err != nil {
		return
	}
	result.Course, err = utils.UnmarshallFile[schema.Course](filepath.Join(dir, "course.json"))
	if err != nil {
		return
	}
	result.Professors, err = utils.UnmarshallFile[[]schema.Professor](filepath.Join(dir, "professors.json"))
	if err != nil {
		return
	}
	result.ClassInfo, err = utils.UnmarshallFile[map[string]string](filepath.Join(dir, "classInfo.json"))
	if err != nil {
		return
	}

	return
}

// updateTestData regenerates /testdata/coursebook/ by parsing all `.html` files under it
// (recursively via utils.GetAllFilesWithExtension) and saving the current
// output as the new expected output.
//
// The expected format for each test case is:
//
//	/case_XXX/
//	  - input.html
//	  - classInfo.json
//	  - course.json
//	  - section.json
//	  - professors.json
//
// It also regenerates the cumulative JSONs (e.g., Courses.json) by running
// Parse on the /testdata/coursebook.
//
// Creates a sample profiles.json by copying professors.json as there is not a good
// why to create it since scrape profiles does not output any intermediate data.
//
// The function creates the new testdata in a temp dir, then replaces the
// existing one atomically to avoid corruption. Duplicate inputs (based on
// SHA-256) are skipped.
//
// Errors may still occur while copying or deleting the testdata directory.
func updateTestData() error {
	//suppress logs
	log.SetOutput(&bytes.Buffer{})
	defer log.SetOutput(os.Stdout)

	tempDir, err := os.MkdirTemp("", "testdata-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	//Fill temp dir with all the test cases and expected values
	duplicates := make(map[string]bool)

	for i, input := range utils.GetAllFilesWithExtension("testdata/coursebook/", ".html") {

		//manually load grades and profiles since it is usually called in Parse()
		if err := loadGrades("testdata/grade-data/"); err != nil {
			return fmt.Errorf("faild to load grade data: %v", err)
		}

		parse(input)

		for _, course := range Courses {
			ReqParsers[course.Id]()
		}

		htmlBytes, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("failed to load test data: %v", err)
		}

		//ensure no duplicate inputs
		hash := sha256.Sum256(htmlBytes)
		hashStr := hex.EncodeToString(hash[:])
		if duplicate := duplicates[hashStr]; duplicate {
			log.Printf("Duplicate test found %s, skipping", input)
			continue
		} else {
			duplicates[hashStr] = true
		}

		//This is gross, the parseXYZ() functions don't return so the only way to access the results is from the maps
		var course schema.Course
		for _, c := range Courses {
			course = *c
			break
		}

		var section schema.Section
		for _, s := range Sections {
			section = *s
			break
		}

		professors := make([]schema.Professor, 0, len(Professors))
		for _, prof := range Professors {
			professors = append(professors, *prof)
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
		if err != nil {
			return fmt.Errorf("failed to parse HTML: %v", err)
		}
		classInfo := getClassInfo(doc)

		caseDir := filepath.Join(tempDir, fmt.Sprintf("case_%03d", i))
		if err = os.Mkdir(caseDir, 0777); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}

		//copy current input.html to individual folder
		if err = os.WriteFile(filepath.Join(caseDir, "input.html"), htmlBytes, 0777); err != nil {
			return fmt.Errorf("failed to write test data: %v", err)
		}

		if err = utils.WriteJSON(filepath.Join(caseDir, "course.json"), course); err != nil {
			return fmt.Errorf("failed to write course %v: %v", course.Id, err)
		}

		if err = utils.WriteJSON(filepath.Join(caseDir, "classInfo.json"), classInfo); err != nil {
			return fmt.Errorf("failed to write class info %v", err)
		}

		if err = utils.WriteJSON(filepath.Join(caseDir, "section.json"), section); err != nil {
			return fmt.Errorf("failed to write section %v: %v", section.Id, err)
		}

		if err = utils.WriteJSON(filepath.Join(caseDir, "professors.json"), professors); err != nil {
			return fmt.Errorf("failed to write professors %v", err)
		}

		//reset all the maps, this is important since we are depend on them to only contain the current set
		clearGlobals()
	}

	//rerun parser to get Courses.json, Sections.json, Professors.json
	Parse(tempDir, tempDir, "testdata/grade-data/", true)

	//overwrite the current test data with the new data
	if err := os.RemoveAll("testdata/coursebook/"); err != nil {
		return fmt.Errorf("failed to remove testdata: %v", err)
	}

	if err := os.CopyFS("testdata/coursebook/", os.DirFS(tempDir)); err != nil {
		return fmt.Errorf("failed to copy testdata: %v", err)
	}

	// since profiles.json is just a list of professor we can just use a list we already have
	// could be better but the profiles scraper also does the parsing for some reason so there
	// is no way to load html like we do for coursebook
	professors, err := utils.UnmarshallFile[[]schema.Professor]("testdata/coursebook/professors.json")
	if err != nil {
		return fmt.Errorf("failed to load professors: %v", err)
	}
	// we need to remove meetings and sections since they are always empty when scrapped
	for i := range professors {
		professors[i].Sections = []primitive.ObjectID{}
		professors[i].Office_hours = []schema.Meeting{}
	}

	if err = utils.WriteJSON("testdata/coursebook/profiles.json", professors); err != nil {
		return fmt.Errorf("failed to create profiles.json: %v", err)
	}

	return nil
}

func clearGlobals() {
	Sections = make(map[primitive.ObjectID]*schema.Section)
	Courses = make(map[string]*schema.Course)
	Professors = make(map[string]*schema.Professor)
	CourseIDMap = make(map[primitive.ObjectID]string)
	ProfessorIDMap = make(map[primitive.ObjectID]string)
	ReqParsers = make(map[primitive.ObjectID]func())
}

func TestParse(t *testing.T) {
	// make sure we are starting from a clean state
	clearGlobals()
	tempDir := t.TempDir()

	Parse("testdata/coursebook/", tempDir, "testdata/grade-data/", false)

	OutputCourses, err := utils.UnmarshallFile[[]schema.Course](filepath.Join(tempDir, "courses.json"))
	if err != nil {
		t.Errorf("failded to load output courses.json %v", err)
	}

	OutputProfessors, err := utils.UnmarshallFile[[]schema.Professor](filepath.Join(tempDir, "professors.json"))
	if err != nil {
		t.Errorf("failded to load output professors.json %v", err)
	}

	OutputSections, err := utils.UnmarshallFile[[]schema.Section](filepath.Join(tempDir, "sections.json"))
	if err != nil {
		t.Errorf("failded to load output sections.json %v", err)
	}

	//Build the ValueByID maps, this is used to for comparing because we cant directly compare ids
	CoursesById := make(map[primitive.ObjectID]schema.Course)
	for _, course := range OutputCourses {
		CoursesById[course.Id] = course
	}
	for _, course := range testCourses {
		CoursesById[course.Id] = course
	}

	ProfessorsByID := make(map[primitive.ObjectID]schema.Professor)
	for _, prof := range OutputProfessors {
		ProfessorsByID[prof.Id] = prof
	}
	for _, prof := range testProfessors {
		ProfessorsByID[prof.Id] = prof
	}

	SectionsByID := make(map[primitive.ObjectID]schema.Section)
	for _, section := range OutputSections {
		SectionsByID[section.Id] = section
	}
	for _, section := range testSections {
		SectionsByID[section.Id] = section
	}

	// check courses
	CoursesByKey := make(map[string]schema.Course)
	//output in to map
	for _, course := range OutputCourses {
		//same key as used in courseParser.go
		key := course.Course_number + course.Catalog_year
		CoursesByKey[key] = course
	}

	for _, expectedCourse := range testCourses {
		key := expectedCourse.Course_number + expectedCourse.Catalog_year
		t.Run(key, func(t *testing.T) {
			if outputCourse, ok := CoursesByKey[key]; ok {
				diff := cmp.Diff(expectedCourse, outputCourse,
					cmpopts.IgnoreFields(schema.Course{}, "Id"),
					cmp.Transformer("Sections", func(sections []primitive.ObjectID) []string {
						result := make([]string, 0, len(sections))
						for _, id := range sections {
							if section, ok := SectionsByID[id]; ok {
								//We don't need to check sections for correctness, just check that the reference is correct
								result = append(result, section.Section_number)
							}
						}
						return result
					}),
				)

				if diff != "" {
					t.Errorf("Course %s mismatch (-expected +got):\n%s", key, diff)
				}

				//remove found course from map, this will allow us to see if there are extra courses in output
				delete(CoursesByKey, key)
			} else {
				t.Errorf("Expected course %s not found in output", key)
			}
		})

	}

	if len(CoursesByKey) > 0 {
		var builder strings.Builder

		builder.WriteString(fmt.Sprintf("Found %d extra Course(s)\n", len(CoursesByKey)))

		for _, course := range CoursesByKey {
			courseText, _ := json.MarshalIndent(course, "", "\t")
			builder.WriteString(string(courseText))
			builder.WriteString("\n")
		}
		t.Error(builder.String())
	}

	//check professors
	ProfessorsByKey := make(map[string]schema.Professor)

	for _, professor := range OutputProfessors {
		//same key as used in professorParser.go
		key := professor.First_name + professor.Last_name
		ProfessorsByKey[key] = professor
	}

	for _, expectedProfessor := range testProfessors {
		key := expectedProfessor.First_name + expectedProfessor.Last_name
		t.Run(key, func(t *testing.T) {

			if outputProfessor, ok := ProfessorsByKey[key]; ok {

				diff := cmp.Diff(expectedProfessor, outputProfessor,
					cmpopts.IgnoreFields(schema.Professor{}, "Id"),
					cmp.Transformer("Sections", func(sections []primitive.ObjectID) []string {
						result := make([]string, 0, len(sections))
						for _, id := range sections {
							if section, ok := SectionsByID[id]; ok {
								//We don't need to check sections for correctness, just check that the reference is correct
								result = append(result, section.Section_number)
							}
						}
						return result
					}),
				)

				if diff != "" {
					t.Errorf("Professor %s mismatch (-expected +got):\n%s", key, diff)
				}
				delete(ProfessorsByKey, key)

			} else {
				t.Errorf("Expected professor %s not found in output", key)
			}
		})
	}

	if len(ProfessorsByKey) > 0 {
		var builder strings.Builder

		builder.WriteString(fmt.Sprintf("Found %d extra Professor(s)\n", len(ProfessorsByKey)))

		for _, course := range ProfessorsByKey {
			courseText, _ := json.MarshalIndent(course, "", "\t")
			builder.WriteString(string(courseText))
			builder.WriteString("\n")
		}
		t.Error(builder.String())
	}

	//check sections
	SectionsByKey := make(map[string]schema.Section)

	for _, section := range OutputSections {
		//the ok shouldn't fail since this is after we checked all the courses
		course := CoursesById[section.Course_reference]
		key := course.Course_number + section.Section_number + section.Academic_session.Name
		SectionsByKey[key] = section
	}

	for _, expectedSection := range testSections {

		course := CoursesById[expectedSection.Course_reference]
		key := course.Course_number + expectedSection.Section_number + expectedSection.Academic_session.Name
		t.Run(key, func(t *testing.T) {
			if outputSection, ok := SectionsByKey[key]; ok {

				diff := cmp.Diff(expectedSection, outputSection,
					cmpopts.IgnoreFields(schema.Section{}, "Id"),
					cmp.Transformer("Course_reference", func(id primitive.ObjectID) string {
						if c, ok := CoursesById[id]; ok {
							return c.Course_number + c.Catalog_year
						}
						return ""
					}),
					cmp.Transformer("Professors", func(profIds []primitive.ObjectID) []string {
						result := make([]string, 0, len(profIds))
						for _, id := range profIds {
							if professor, ok := ProfessorsByID[id]; ok {
								//We don't need to check sections for correctness, just check that the reference is correct
								result = append(result, professor.First_name+professor.Last_name)
							}
						}
						return result
					}),
				)

				if diff != "" {
					t.Errorf("Section %s mismatch (-expected +got):\n%s", key, diff)
				}

				delete(SectionsByKey, key)
			} else {
				t.Errorf("Expected Section %s not found in output", key)
			}
		})
	}

	if len(SectionsByKey) > 0 {
		var builder strings.Builder

		builder.WriteString(fmt.Sprintf("Found %d extra Sections(s) : \n", len(SectionsByKey)))

		for _, section := range SectionsByKey {
			courseText, _ := json.MarshalIndent(section, "", "\t")
			builder.WriteString(string(courseText))
			builder.WriteString("\n")
		}
		t.Error(builder.String())
	}
}

func TestGetClassInfo(t *testing.T) {
	t.Parallel()

	for name, testCase := range testData {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(testCase.Input))
			if err != nil {
				return
			}
			output := getClassInfo(doc)
			expected := testCase.ClassInfo

			diff := cmp.Diff(expected, output)
			if diff != "" {
				t.Errorf("Failed (-expected +got)\n %s", diff)
			}
		})

	}
}

func TestGetRowInfo(t *testing.T) {
	t.Parallel()
	// don't include any weird characters in the content, it's not a bug with getRowInfo but
	// goquery will modify content when encoding/decoding html so the result will not match content.
	testCases := map[string]struct {
		Title   string
		Content string
	}{
		"case_001": {
			Title:   "Course Title:",
			Content: "Introductory Financial Accounting",
		},
		"case_002": {
			Title:   "Evaluation:",
			Content: "<i>An evaluation report for Introductory Financial Accounting  (ACCT2301.003.25S) has not been posted.</i>",
		},
		"case_003": {
			Title:   "Schedule:",
			Content: "<a href=\"https://dox.utdallas.edu/syl152555\" target=\"_blank\">Syllabus for Introductory Financial Accounting  (ACCT2301.003.25S)</a>",
		},
		"case_004": {
			Title:   "Class Info:",
			Content: "<table class=\"courseinfo__classsubtable\"><tbody></tbody></table>",
		},
		"case_005": {
			Title:   "",
			Content: "",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			html := fmt.Sprintf(`
				<div class="expandblock-content">
        			<table class="courseinfo__overviewtable">
            		<tbody>
							<tr class="courseinfo__overviewtable__tr">
                				<th class="courseinfo__overviewtable__th text-right">%s</th>
                				<td class="courseinfo__overviewtable__td courseinfo__overviewtable__coursetitle">%s</td>
            				</tr>
						</tbody>
        			</table>
    			</div>`, testCase.Title, testCase.Content)

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				t.Fatalf("failed to create document: %v", err)
			}

			rowInfo := getRowInfo(doc)

			if row, ok := rowInfo[utils.TrimWhitespace(testCase.Title)]; ok {
				content, err := row.Html()
				if err != nil {
					t.Fatalf("failed to get row content: %v", err)
				}

				if diff := cmp.Diff(testCase.Content, content); diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
				}
			} else {
				t.Errorf("Failed to find row in infoRows")
			}
		})
	}
}

// utils
func FailTestIfNoPanic(t *testing.T, name string) {
	if r := recover(); r == nil {
		t.Errorf("expected %s to panic but it did not", name)
	}
}

func FailTestIfPanic(t *testing.T, name string) {
	if r := recover(); r != nil {
		t.Errorf("%s failed with error %v", name, r)
	}
}

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

// testData global dictionary containing the data from /testdata by folder name
var testData map[string]TestData

// TestMain entry point for all tests in the parser package.
// The function will load `./testdata` into memory before running
// the tests so that test can run in parallel.
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
		if err := updateTestData(); err != nil {
			log.Fatalf("Error updating test data: %v", err)
		}
		log.Println("Successfully updated test data")
		os.Exit(0)
	}

	testData = make(map[string]TestData)
	dir, err := os.ReadDir("testdata")
	if err != nil {
		log.Fatalf("Failed to load testdata: %v", err)
	}

	for _, file := range dir {
		if !file.IsDir() {
			continue
		}
		if testData[file.Name()], err = loadTest(file.Name()); err != nil {
			log.Fatalf("Failed to load %s: %v", file.Name(), err)
		}
	}

	os.Exit(m.Run())
}

func loadTest(dir string) (result TestData, err error) {
	htmlBytes, err := os.ReadFile(fmt.Sprintf("testdata/%s/input.html", dir))
	if err != nil {
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return
	}
	result.Input = string(htmlBytes)
	result.RowInfo = getRowInfo(doc)
	result.Section, err = unmarshallFile[schema.Section](fmt.Sprintf("testdata/%s/section.json", dir))
	if err != nil {
		return
	}
	result.Course, err = unmarshallFile[schema.Course](fmt.Sprintf("testdata/%s/course.json", dir))
	if err != nil {
		return
	}
	result.Professors, err = unmarshallFile[[]schema.Professor](fmt.Sprintf("testdata/%s/professors.json", dir))
	if err != nil {
		return
	}
	result.ClassInfo, err = unmarshallFile[map[string]string](fmt.Sprintf("testdata/%s/classinfo.json", dir))
	if err != nil {
		return
	}

	return
}

// updateTestData regenerates /testdata by parsing all `.html` files under it
// (recursively via utils.GetAllFilesWithExtension) and saving the current
// output as the new expected output.
//
// The expected format for each test case is:
//
//	/case_XXX/
//	  - input.html
//	  - classinfo.json
//	  - course.json
//	  - section.json
//	  - professors.json
//
// It also regenerates the cumulative JSONs (e.g., Courses.json) by running
// Parse on the /testdata.
//
// The function creates the new testdata in a temp dir, then replaces the
// existing one atomically to avoid corruption. Duplicate inputs (based on
// SHA-256) are skipped.
//
// Errors may still occur while copying or deleting the testdata directory.
func updateTestData() error {
	log.Printf("Updating test data for the given inputs")
	//doesn't do anything since there is no profile data
	loadProfiles("")

	tempDir, err := os.MkdirTemp("", "testdata-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	//Fill temp dir with all the test cases and expected values

	duplicates := make(map[string]bool)

	for i, input := range utils.GetAllFilesWithExtension("testdata", ".html") {
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
			log.Printf("Duplicate test found %s, slipping\n", input)
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

		if err = utils.WriteJSON(filepath.Join(caseDir, "ClassInfo.json"), classInfo); err != nil {
			return fmt.Errorf("failed to write class info %v", err)
		}

		if err = utils.WriteJSON(filepath.Join(caseDir, "section.json"), section); err != nil {
			return fmt.Errorf("failed to write section %v: %v", section.Id, err)
		}

		if err = utils.WriteJSON(filepath.Join(caseDir, "professors.json"), professors); err != nil {
			return fmt.Errorf("failed to write professors %v", err)
		}

		//reset all the maps, this is important since we are depending on them to only contain the current set
		clearGlobals()
	}

	//rerun parser to get Courses.json, Sections.json, Professors.json

	//Parse(tempDir, tempDir, "../grade-data", false)
	//Grade data isn't work with tests currently
	Parse(tempDir, tempDir, "", false)

	//overwrite the current test data with the new data
	if err := os.RemoveAll("testdata"); err != nil {
		return fmt.Errorf("failed to remove testdata: %v", err)
	}

	if err := os.CopyFS("testdata", os.DirFS(tempDir)); err != nil {
		return fmt.Errorf("failed to copy testdata: %v", err)
	}

	//reset maps to avoid side effects. maybe parser should be an object?
	clearGlobals()
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
	tempDir := t.TempDir()
	// todo fix grade data, csvPath = ./grade-data panics
	Parse("testdata", tempDir, "", false)

	OutputCourses, err := unmarshallFile[[]schema.Course](filepath.Join(tempDir, "courses.json"))
	if err != nil {
		t.Errorf("failded to load output courses.json %v", err)
	}

	OutputProfessors, err := unmarshallFile[[]schema.Professor](filepath.Join(tempDir, "professors.json"))
	if err != nil {
		t.Errorf("failded to load output professors.json %v", err)
	}

	OutputSections, err := unmarshallFile[[]schema.Section](filepath.Join(tempDir, "sections.json"))
	if err != nil {
		t.Errorf("failded to load output sections.json %v", err)
	}

	ExpectedCourses, err := unmarshallFile[[]schema.Course](filepath.Join("testdata", "courses.json"))
	if err != nil {
		t.Errorf("failded to load expected courses.json %v", err)
	}

	ExpectedProfessors, err := unmarshallFile[[]schema.Professor](filepath.Join("testdata", "professors.json"))
	if err != nil {
		t.Errorf("failded to load expected professors.json %v", err)
	}

	ExpectedSections, err := unmarshallFile[[]schema.Section](filepath.Join("testdata", "sections.json"))
	if err != nil {
		t.Errorf("failded to load expected sections.json %v", err)
	}

	//Build the ValueByID maps, this is used to for comparing because we cant directly compare ids
	CoursesById := make(map[primitive.ObjectID]schema.Course)
	for _, course := range OutputCourses {
		CoursesById[course.Id] = course
	}
	for _, course := range ExpectedCourses {
		CoursesById[course.Id] = course
	}

	ProfessorsByID := make(map[primitive.ObjectID]schema.Professor)
	for _, prof := range OutputProfessors {
		ProfessorsByID[prof.Id] = prof
	}
	for _, prof := range ExpectedProfessors {
		ProfessorsByID[prof.Id] = prof
	}

	SectionsByID := make(map[primitive.ObjectID]schema.Section)
	for _, section := range OutputSections {
		SectionsByID[section.Id] = section
	}
	for _, section := range ExpectedSections {
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

	for _, expectedCourse := range ExpectedCourses {
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
							} else {
								result = append(result, "")
							}
						}
						return result
					}),
				)

				if diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
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

	for _, expectedProfessor := range ExpectedProfessors {
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
							} else {
								result = append(result, "")
							}
						}
						return result
					}),
				)

				if diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
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
		key := course.Course_number + course.Catalog_year + section.Section_number
		SectionsByKey[key] = section
	}

	for _, expectedSection := range ExpectedSections {

		course := CoursesById[expectedSection.Course_reference]
		key := course.Course_number + course.Catalog_year + expectedSection.Section_number
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
							} else {
								result = append(result, "")
							}
						}
						return result
					}),
				)

				if diff != "" {
					t.Errorf("Failed (-expected +got)\n %s", diff)
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

// unmarshallFile reads a JSON file from the given path and unmarshals it into type T.
func unmarshallFile[T any](path string) (T, error) {
	var result T

	file, err := os.ReadFile(path)
	if err != nil {
		return result, fmt.Errorf("error reading file '%s': %w", path, err) // Wrap original error
	}
	if err = json.Unmarshal(file, &result); err != nil {
		return result, fmt.Errorf("error unmarshalling JSON from file '%s': %w", path, err) // Wrap original error
	}

	return result, nil
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

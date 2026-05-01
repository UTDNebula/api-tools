/*
Code requires having pdftotext installed: https://www.xpdfreader.com/pdftotext-man.html
apt-get install -y poppler-utils
I found all the Go programs for PDF text extraction were all either paid, had a
complicated installation process, or errored on one of the PDFs.
*/

package parser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
	"google.golang.org/genai"
)

// What gets sent to Gemini, with the PDF content added
var budgetPrompt = `Parse the content of these PDFs and generate the following JSON schema.

{
  _id: %s,
  operating_budget: { // Data only from the Operating Budget file
    operating_revenues: {
      name: "Operating Revenues", // From the Operating Budget - Expenses by Functional Classification table
      rows: [
        {
          label: string, // Tuition and Fees (Gross), Less Discounts and Allowances, Federal Sponsored Programs, ...
          value: number // Use the total from the latest FY (last column)
        }
      ],
      total: number
    },
    operating_expenses: {
      name: "Operating Expenses", // Right under, from the same Operating Budget - Expenses by Functional Classification table
      rows: [
        {
          label: string, // Instruction, Academic Support, Research, ...
          value: number // Use the total from the latest FY (last column)
        }
      ],
      total: number
    },
    budgeted_nonoperating_revenues: {
      name: "Budgeted Nonoperating Revenues (Expenses)", // Right under, from the same Operating Budget - Expenses by Functional Classification table
      rows: [
        {
          label: string, // State Appropriations, Federal Sponsored Programs - Nonoperating, State/Local Sponsored Programs - Nonoperating, ...
          value: number // Use the total from the latest FY (last column)
        }
      ],
      total: number
    },
		salaries_doe_and_instructional_admin: {
      name: "Summary of Faculty Salaries, Departmental Operating Expenses, and Instructional Administration",
      rows: [
        {
          label: string, // Provost and V.P. Academic Affairs, each school, Other Instructional Support
          value: { // Use the values from the latest FY (last 4 columns)
						total: number,
						faculty_salaries: number,
						departmental_operating_expenses: number,
						instructional_administration: number
					}
        }
      ],
      total: {
				total: number,
				faculty_salaries: number,
				departmental_operating_expenses: number,
				instructional_administration: number
			}
    },
		service_departments_funds: {
      name: "Service Departments and Revolving Funds", // In the Service Department Funds section
      rows: [
        {
          name: string, // Sub tables by school and other categories
          rows: [
            {
              label: string,
              value: {
                estimated_income: number,
                budgeted_expenses: number
              }
            }
          ],
          total: {
            estimated_income: number,
            budgeted_expenses: number
          }
        }
      ],
      total: {
        estimated_income: number,
        budgeted_expenses: number
      }
    },
		designated_funds: {
      name: "Designated Funds", // In the Designated Funds section
      rows: [
        {
          name: string, // Sub tables by school and other categories
          rows: [
            {
              label: string,
              value: {
                estimated_income: number,
                budgeted_expenses: number
              }
            }
          ],
          total: {
            estimated_income: number,
            budgeted_expenses: number
          }
        }
      ],
      total: {
        estimated_income: number,
        budgeted_expenses: number
      }
    },
    budgeted_tuition_and_student_fees: {
      name: "Budgeted Tuition and Student Fees", // In the Designated Funds section
      rows: [
        {
          name: string, // Tuition, Laboratory & Supplemental Fees, Mandatory Student Fees, Program, Course Related & Other Incidental Fees
          rows: [
            {
              label: string, // Tuition, Tuition Differential Exemption; Laboratory Fees Excessive Hours, Three Peat Fee; Advising Fee, Athletic Program Fee, Information Technology Fee; Application Fee; Bursar Fees, Late Fees; Chec Collin County
              value: number // Use the total from the latest FY (last column)
            }
          ],
          total: number
        }
      ],
      total: number
    },
    auxiliary_expenses: {
      name: "Auxiliary Expenses", // In the Auxiliary Enterprises Funds section
      rows: [
        {
          name: string, // Sub tables by school and other categories including Facilities and Economic Dev, Student Affairs, ...
          rows: [
            {
              label: string,
              value: {
                estimated_income: number,
                budgeted_expenses: number,
                debt_service: number,
								other: number
              }
            }
          ],
          total: {
            estimated_income: number,
            budgeted_expenses: number,
            debt_service: number,
						other: number
          }
        }
      ],
      total: {
        estimated_income: number,
        budgeted_expenses: number,
        debt_service: number,
				other: number
      }
    },
		restricted_funds: {
      name: "Restricted Funds", // In the Restricted Gift Funds section, Endowments table
      rows: [
        {
          name: string, // Sub tables by school and other categories
          rows: [
            {
              label: string,
              value: {
                estimated_income: number,
                budgeted_expenses: number
              }
            }
          ],
          total: {
            estimated_income: number,
            budgeted_expenses: number
          }
        }
      ],
      total: {
        estimated_income: number,
        budgeted_expenses: number
      }
    }
  },
  annual_financial_report: { // Data only from the Annual Financial Report file
		// All from the Exhibit B Statement of Revenues, Expenses, and Changes in Net Position table
    operating_revenues: {
      name: "Operating Revenues",
      rows: [
        {
          label: string, // Student Tuition and Fees, Discounts and Allowances, Federal Sponsored Programs, ...
          value: number // Use the total from the latest FY (first column)
        }
      ],
      total: number
    },
    operating_expenses: {
      name: "Operating Expenses",
      rows: [
        {
          label: string, // Instruction, Research, Public Service, ...
          value: number // Use the total from the latest FY (first column)
        }
      ],
      total: number
    },
    nonoperating_revenues: {
      name: "Nonoperating Revenues (Expenses)",
      rows: [
        {
          label: string, // State Appropriations, Federal Nonexchange Sponsored Programs, Federal Nonexchange Pass-Through, ...
          value: number // Use the total from the latest FY (first column)
        }
      ],
      total: number
    },
    beginning_net_position: number,
    ending_net_position: number
  }
}

- Use the full UTD school names in this title text: School of Arts, Humanities, and Technology; School of Behavioral and Brain Sciences; School of Economic, Political and Policy Sciences; School of Engineering and Computer Science; School of Interdisciplinary Studies; School of Management; School of Natural Sciences and Mathematics.
  - In older years: School of Arts, Technology, and Emerging Communication; School of Arts & Humanities.
	- Replace Brian with Brain in the School of Behavioral and Brain Sciences name if it is misspelled in the PDF.
- Always use the data listed for %s, not any previous years.
- Do not infer, estimate, or guess any values. 
- If a value is missing or unclear, return null for that field.
- Only values surrounded by parentheses in the tables should be considered negative.

Content of PDFs:

%s`

func ParseBudgets(inDir string, outDir string, budgetsDir string, useBackupBudgets bool) {
	// Get sub folder from output folder
	inSubDir := filepath.Join(inDir, "budgets")

	result := []schema.Budget{}

	// Parallel requests
	numWorkers := 10
	jobs := make(chan []string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start worker goroutines
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for paths := range jobs {
				year := filepath.Base(filepath.Dir(paths[0]))
				log.Printf("Parsing %s...", year)

				budget, err := parseBudgetPdfs(paths)
				if err != nil {
					if strings.Contains(err.Error(), "429") {
						// Exponential-ish backoff up to 60s for 429 rate limiting
						backoffs := []time.Duration{20 * time.Second, 40 * time.Second, 60 * time.Second}
						for _, delay := range backoffs {
							time.Sleep(delay)
							budget, err = parseBudgetPdfs(paths)
							if err == nil || !strings.Contains(err.Error(), "429") {
								break
							}
						}
					}

					if err != nil {
						panic(err)
					}
				}

				mu.Lock()
				result = append(result, budget)
				mu.Unlock()

				log.Printf("Parsed %s!", year)
			}
		}()
	}

	yearsScraped := make(map[string]bool)

	// Traverse scraped directory, storing years found
	err := filepath.WalkDir(inSubDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root "budgets" directory itself
		if path == inSubDir {
			return nil
		}
		if d.IsDir() { // Is a folder
			year := filepath.Base(path)
			yearsScraped[year] = true
			var files []string
			err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() { // Is a file
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return err
			}
			jobs <- files
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	// If we're including the backup budgets, since UTD has removed some older PDFs from the website
	if useBackupBudgets {
		// Traverse backup directory, only adding jobs for years not found in scraped directory
		err = filepath.WalkDir(budgetsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Skip the root "budgets" directory itself
			if path == budgetsDir {
				return nil
			}
			if d.IsDir() { // Is a folder
				year := filepath.Base(path)
				if !yearsScraped[year] {
					var files []string
					err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
						if err != nil {
							return err
						}
						if !d.IsDir() { // Is a file
							files = append(files, path)
						}
						return nil
					})
					if err != nil {
						return err
					}
					jobs <- files
				}
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}

	close(jobs)

	// Wait for workers to finish
	wg.Wait()

	utils.WriteJSON(fmt.Sprintf("%s/budgets.json", outDir), result)
}

// Read a PDF, build a prompt for Gemini to parse it, check if it has already been asked in the cache, and ask Gemini if not
func parseBudgetPdfs(paths []string) (schema.Budget, error) {
	apiBucket, err := getBudgetBucket()
	if err != nil {
		return schema.Budget{}, err
	}

	year := filepath.Base(filepath.Dir(paths[0]))

	// Read PDFs
	var contentBuilder strings.Builder
	for _, path := range paths {
		content, err := utils.ReadPdf(path, -1)
		if err != nil {
			return schema.Budget{}, err
		}
		contentBuilder.WriteString("# " + filepath.Base(path) + "\n\n")
		contentBuilder.WriteString(content + "\n\n\n")
	}
	content := contentBuilder.String()

	// Build prompt
	promptFilled := fmt.Sprintf(budgetPrompt, year, year, content)

	// Check cache
	hashByte := sha256.Sum256([]byte(promptFilled))
	hash := hex.EncodeToString(hashByte[:]) + ".json"
	result, err := utils.CheckCache(hash, apiBucket)
	if err != nil {
		return schema.Budget{}, err
	}

	// Skip AI if cache found
	if result != "" {
		log.Printf("Cache found for %s!", year)
	} else {
		// Cache not found
		log.Printf("No cache for %s, asking Gemini.", year)

		// AI
		geminiClient := utils.GetGeminiClient()

		// Response schema
		budgetSchema := utils.StructToSchema(reflect.TypeOf(schema.Budget{}))

		// Send request with default config
		response, err := geminiClient.Models.GenerateContent(context.Background(),
			"gemini-2.5-pro",
			genai.Text(promptFilled),
			// Enforce response schema
			&genai.GenerateContentConfig{
				ResponseMIMEType: "application/json",
				ResponseSchema:   budgetSchema,
			},
		)
		if err != nil {
			return schema.Budget{}, err
		}

		// Get response
		result = response.Candidates[0].Content.Parts[0].Text
		log.Print("Token counts:")
		log.Printf("Prompt: %d", response.UsageMetadata.PromptTokenCount)
		log.Printf("Thoughts: %d", response.UsageMetadata.ThoughtsTokenCount)
		log.Printf("Total: %d", response.UsageMetadata.TotalTokenCount)

		// Set cache for next time
		err = utils.SetCache(hash, result, apiBucket)
		if err != nil {
			return schema.Budget{}, err
		}
	}

	// Build struct
	var budget schema.Budget
	err = json.Unmarshal([]byte(result), &budget)
	if err != nil {
		return schema.Budget{}, err
	}

	return budget, nil
}

// Get the storage bucket for the budget cache
func getBudgetBucket() (string, error) {
	apiBucket, err := utils.GetEnv("NEBULA_API_BUDGET_STORAGE_BUCKET")
	if err != nil {
		return "", err
	}
	return apiBucket, nil
}

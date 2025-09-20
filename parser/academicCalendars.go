package parser

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/ledongthuc/pdf"
)

func ParseAcademicCalendars(inDir string, outDir string) {
	pdf.DebugOn = true

	// Get sub folder from output folder
	outSubDir := filepath.Join(outDir, "academicCalendars")

	err := filepath.WalkDir(outSubDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() { // Is a file
			err = parsePdf(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

func readPdf(path string) (string, error) {
	// Open the PDF
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Read plain text
	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		panic(err)
	}
	buf.ReadFrom(b)
	content := buf.String()

	return content, nil
}

func parsePdf(path string) error {
	fmt.Println("Parsing:", path)

	content, err := readPdf(path)
	if err != nil {
		return err
	}
	fmt.Println("\n\n\nExtracted text:\n", content)

	return nil
}

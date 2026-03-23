package uploader

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/nebula-api/api/schema"
)

const DEGREES_FILE string = "degrees.json"

func UploadDegrees(inDir string) {
	client := connectDBFunc()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	// Open data file for reading
	fptr, err := os.Open(fmt.Sprintf("%s/"+DEGREES_FILE, inDir))
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File not found. Skipping %s", DEGREES_FILE)
			return
		}
		log.Panic(err)
	}
	defer fptr.Close()

	UploadData[schema.AcademicProgram](client, ctx, fptr, true)
}

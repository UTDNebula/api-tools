/*
	This file is responsible for handling uploading of parsed discount programs data to MongoDB.
*/

package uploader

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/UTDNebula/nebula-api/api/schema"
)

//  Note that this uploader assumes that the collection names match the names of these files, which they should.
//  If the names of these collections ever change, the file names should be updated accordingly.

var discountsFile = "discounts.json"

func UploadDiscounts(inDir string) {
	//Connect to mongo
	client := connectDBFunc()

	// Get 5 minute context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Open data file for reading
	fptr, err := os.Open(fmt.Sprintf("%s/"+discountsFile, inDir))
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File not found. Skipping %s", discountsFile)
			return
		}
		log.Panic(err)
	}

	defer fptr.Close()

	UploadData[schema.DiscountProgram](client, ctx, fptr, true)
}

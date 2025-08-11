/*
	This file contains the code for the letters generator.
*/

package generators

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/UTDNebula/api-tools/utils"
	"github.com/UTDNebula/nebula-api/api/schema"
)

var NUM_LETTERS int = 21

func GenerateLetters(outDir string) {
	// Make output folder
	err := os.MkdirAll(outDir, 0777)
	if err != nil {
		panic(err)
	}

	// Define tiles and weights: https://en.wikipedia.org/wiki/Scrabble_letter_distributions
	tiles := []rune{
		'K', 'J', 'X', 'Q', 'Z', // x1
		'B', 'C', 'M', 'P', 'F', 'H', 'V', 'W', 'Y', // x2
		'G',                // x3
		'L', 'S', 'U', 'D', // x4
		'N', 'R', 'T', // x6
		'O',      // x8
		'A', 'I', // x9
		'E', // x12
	}
	weights := []int{
		1, 1, 1, 1, 1, // x1
		2, 2, 2, 2, 2, 2, 2, 2, 2, // x2
		3,          // x3
		4, 4, 4, 4, // x4
		6, 6, 6, // x6
		8,    // x8
		9, 9, // x9
		12, // x12
	}

	// Seed random number generator
	localRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Precompute cumulative distribution
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}
	cumulative := make([]int, len(weights))
	sum := 0
	for i, w := range weights {
		sum += w
		cumulative[i] = sum
	}

	// Function to draw a random tile based on weights
	drawTile := func() rune {
		r := localRand.Intn(totalWeight) + 1
		for i, c := range cumulative {
			if r <= c {
				return tiles[i]
			}
		}
		return tiles[len(tiles)-1] // fallback
	}

	// Draw NUM_LETTERS random tiles
	result := make([]rune, NUM_LETTERS)
	for i := 0; i < NUM_LETTERS; i++ {
		result[i] = drawTile()
	}

	// Get the date
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		log.Fatalf("Error loading location: %v", err)
	}
	utcNow := time.Now().UTC()
	today := utcNow.In(loc).Format("2006-01-02")

	// Format output
	output := []schema.Letters{{
		Date:    today,
		Letters: string(result),
	}}

	log.Print("Generated letters!")

	// Write letters to output file
	utils.WriteJSON(fmt.Sprintf("%s/letters.json", outDir), output)
}

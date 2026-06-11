package scrapers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/UTDNebula/api-tools/utils"
)

func ScrapeEvals(term string, outDir string) {
	scraper := newCoursebookScraper(term, outDir)
	defer scraper.chromedpCancel()

	prefixes := utils.GetCoursePrefixes(scraper.chromedpCtx)

	for _, prefix := range prefixes {
		err := os.MkdirAll(filepath.Join(outDir, "evals", term, prefix), 0755)
		if err != nil {
			log.Panicf("failed to create folder for %s: %s", prefix, err)
		}

		ids, err := scraper.getSectionIdsForPrefix(prefix)
		if err != nil {
			panic(err)
		}

		for _, id := range ids {
			//res, err := scraper.req(fmt.Sprintf("https://coursebook.utdallas.edu/ues-report/%s", id), 3, "evals")
			res, err := scraper.evalReq(id)
			if err != nil {
				panic(err)
			}

			path := filepath.Join(scraper.outDir, "evals", term, prefix, id+".html")

			err = os.WriteFile(path, res, 0644)
			if err != nil {
				log.Panicf("failed to write section %s: %s", id, err)
			}
		}

	}

}

// req utility function for making calling the coursebook api
func (s *coursebookScraper) evalReq(id string) ([]byte, error) {
	var res *http.Response
	retries := 3
	reqName := fmt.Sprintf("GET EVAL:%s", id)

	err := utils.Retry(func() error {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://coursebook.utdallas.edu/ues-report/%s", id), nil)
		if err != nil {
			log.Fatalf("Http request failed: %v", err)
		}
		req.Header = s.coursebookHeaders

		start := time.Now()
		res, err = s.httpClient.Do(req)
		dur := time.Since(start)

		if res != nil {
			if res.StatusCode != 200 {
				return errors.New("non-200 response status code")
			}
			utils.VPrintf("[Request Success] Request for [%s] took %v", reqName, dur)
		} else if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				utils.VPrintf("[Timeout] Request for [%s] timed out", reqName)
			} else {
				utils.VPrintf("[Request Error] Request for %s failed: %v", reqName, err)
			}
		}

		return err
	}, retries, func(numRetries int) {
		utils.VPrintf("[Request Retry] Attempt %d of %d for request %s", numRetries, retries, reqName)
		s.coursebookHeaders = utils.RefreshToken(s.chromedpCtx)
		s.reqRetries++

		//back off exponentially
		time.Sleep(time.Duration(math.Pow(2, float64(numRetries))) * time.Second)
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	content, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}
	return content, nil
}

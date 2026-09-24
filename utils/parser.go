// Package utils provides shared helpers for parsing workflows.
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"sync"

	"strings"

	"github.com/UTDNebula/nebula-api/api/schema"
	"google.golang.org/genai"
)

// Read the text from the first n pages of a PDF
// Using external program pdftotext
// Code requires having pdftotext installed: https://www.xpdfreader.com/pdftotext-man.html
// apt-get install -y poppler-utils
func ReadPdf(path string, lastPage int) (string, error) {
	cmd := exec.Command("pdftotext", "-raw", path, "-")
	if lastPage > 0 {
		cmd = exec.Command("pdftotext", "-l", strconv.Itoa(lastPage), "-raw", path, "-")
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run pdftotext: %v (%s)", err, stderr.String())
	}

	return out.String(), nil
}

// Check cache for a response to the same prompt
func CheckCache(hash string, apiBucket string) (string, error) {
	apiUrl, apiKey, apiStorageKey, err := getNebulaKeys()
	if err != nil {
		return "", err
	}

	client := &http.Client{}

	// Make request
	req, err := http.NewRequest("GET", apiUrl+"storage/"+apiBucket+"/"+hash, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("x-api-key", apiKey)
	req.Header.Add("x-storage-key", apiStorageKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var parsedBody schema.APIResponse[schema.ObjectInfo]
	err = json.Unmarshal([]byte(body), &parsedBody)
	if err != nil {
		// If this errors, return ("", nil) to indicate not found
		return "", nil
	}

	// Fetch object
	req, err = http.NewRequest("GET", parsedBody.Data.MediaLink, nil)
	if err != nil {
		return "", err
	}
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Store Gemini client to only create once
var once sync.Once
var geminiClient *genai.Client

// Create client only once
// Auth is from GOOGLE_GENAI_USE_VERTEXAI, GOOGLE_CLOUD_PROJECT and GOOGLE_APPLICATION_CREDENTIALS environment variables and service account JSON which is created from GEMINI_SERVICE_ACCOUNT
func GetGeminiClient() *genai.Client {
	once.Do(func() {
		// Create JSON file
		serviceAccount, err := GetEnv("GEMINI_SERVICE_ACCOUNT")
		if err != nil {
			panic(err)
		}
		jsonFile, err := GetEnv("GOOGLE_APPLICATION_CREDENTIALS")
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(jsonFile, []byte(serviceAccount), 0644)
		if err != nil {
			panic(err)
		}

		// Create client
		geminiClient, err = genai.NewClient(context.Background(),
			&genai.ClientConfig{
				Project:  "api-tools-451421",
				Location: "us-central1",
				Backend:  genai.BackendVertexAI,
			})
		if err != nil {
			panic(err)
		}
	})
	return geminiClient
}

func StructToSchema(t reflect.Type) *genai.Schema {
	// Handle pointers
	isNullable := false
	if t.Kind() == reflect.Ptr {
		isNullable = true
		t = t.Elem()
	}

	var schema *genai.Schema

	switch t.Kind() {
	case reflect.Struct:
		properties := make(map[string]*genai.Schema)
		var required []string

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			// Use the JSON tag for the property name
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			// Handle comma-separated tags like "name,omitempty"
			name := strings.Split(jsonTag, ",")[0]

			properties[name] = StructToSchema(field.Type)
			if field.Type.Kind() != reflect.Ptr {
				required = append(required, name)
			}
		}

		schema = &genai.Schema{
			Type:       genai.TypeObject,
			Properties: properties,
			Required:   required,
		}

	case reflect.Slice, reflect.Array:
		schema = &genai.Schema{
			Type:  genai.TypeArray,
			Items: StructToSchema(t.Elem()),
		}

	case reflect.String:
		schema = &genai.Schema{Type: genai.TypeString}

	case reflect.Int, reflect.Int64, reflect.Float64:
		schema = &genai.Schema{Type: genai.TypeNumber}

	case reflect.Bool:
		schema = &genai.Schema{Type: genai.TypeBoolean}

	default:
		schema = &genai.Schema{Type: genai.TypeString}
	}

	schema.Nullable = &isNullable
	return schema
}

// Upload AI response to cache
func SetCache(hash string, result string, apiBucket string) error {
	apiUrl, apiKey, apiStorageKey, err := getNebulaKeys()
	if err != nil {
		return err
	}

	// Make request
	jsonStr := []byte(result)
	bodyReader := bytes.NewBuffer(jsonStr)
	req, err := http.NewRequest("POST", apiUrl+"storage/"+apiBucket+"/"+hash, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("x-api-key", apiKey)
	req.Header.Add("x-storage-key", apiStorageKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Get all the keys to access the Nebula API storage routes
func getNebulaKeys() (string, string, string, error) {
	apiUrl, err := GetEnv("NEBULA_API_URL")
	if err != nil {
		return "", "", "", err
	}
	apiKey, err := GetEnv("NEBULA_API_KEY")
	if err != nil {
		return "", "", "", err
	}
	apiStorageKey, err := GetEnv("NEBULA_API_STORAGE_KEY")
	if err != nil {
		return "", "", "", err
	}

	return apiUrl, apiKey, apiStorageKey, nil
}

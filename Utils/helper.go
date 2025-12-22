package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

var PythonServer string
var LocalStorage string

func InitEnv(){
	LocalStorage = os.Getenv("LocalStorage_PATH")
	PythonServer = os.Getenv("PYTHON_SERVER")
}

func GenerateRandomString(length int) string {
	return uuid.New().String()[:length]
}


func DoPost(url string, body map[string]interface{}) (map[string]interface{}, error) {
	// Marshal the body into JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a client with timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle non-200 responses with body included for context
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response %d: %s", resp.StatusCode, string(respBody))
	}

	// If response is empty
	if len(respBody) == 0 {
		return nil, nil
	}

	// Unmarshal JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// Response represents a standardized API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// SendJSONResponse sends a standardized JSON response with proper headers
// Sets Content-Type: application/json and handles encoding consistently
func SendJSONResponse(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Log encoding errors but don't expose to client
		fmt.Printf("Failed to encode JSON response: %v\n", err)
	}
}

// SendErrorResponse sends a standardized error response
// Use this for all error responses to maintain consistency
func SendErrorResponse(w http.ResponseWriter, status int, message string) {
	SendJSONResponse(w, status, Response{
		Success: false,
		Error:   message,
	})
}

// SendSuccessResponse sends a standardized success response
// Use this for all successful responses to maintain consistency
func SendSuccessResponse(w http.ResponseWriter, data interface{}) {
	SendJSONResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

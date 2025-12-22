package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	ESClient    *http.Client
	ESBaseURL   string
	ESEnabled   bool
	UsersIndex  = "users"
	VideosIndex = "videos"
)

// GetESClient returns the Elasticsearch HTTP client
func GetESClient() *http.Client {
	return ESClient
}

// GetESBaseURL returns the Elasticsearch base URL
func GetESBaseURL() string {
	return ESBaseURL
}

// IsESEnabled returns whether Elasticsearch is enabled
func IsESEnabled() bool {
	return ESEnabled
}

// GetUsersIndex returns the users index name
func GetUsersIndex() string {
	return UsersIndex
}

// GetVideosIndex returns the videos index name
func GetVideosIndex() string {
	return VideosIndex
}

// InitElasticsearch initializes the Elasticsearch client
func InitElasticsearch() {
	esHost := os.Getenv("ELASTICSEARCH_HOST")
	esPort := os.Getenv("ELASTICSEARCH_PORT")

	if esHost == "" {
		esHost = "localhost"
	}
	if esPort == "" {
		esPort = "9200"
	}

	ESBaseURL = fmt.Sprintf("http://%s:%s", esHost, esPort)
	ESClient = &http.Client{
		Timeout: 15 * time.Second, // Increased timeout to match context timeout
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("Attempting to connect to Elasticsearch at %s...", ESBaseURL)
	if err := pingElasticsearch(ctx); err != nil {
		log.Printf("Warning: Elasticsearch connection failed: %v. Elasticsearch features will be disabled.", err)
		log.Printf("Make sure Elasticsearch is running at %s", ESBaseURL)
		ESEnabled = false
		return
	}

	ESEnabled = true
	log.Println("Elasticsearch connected!")

	// Create indices if they don't exist (each index gets its own timeout context)
	log.Println("Creating Elasticsearch indices...")
	if err := createIndices(context.Background()); err != nil {
		log.Printf("Warning: Failed to create Elasticsearch indices: %v", err)
		log.Printf("Indices may already exist or Elasticsearch may be slow to respond. You can create them manually if needed.")
		// Don't disable ES - it's connected, just index creation failed
		// Indices might already exist or can be created manually
	} else {
		log.Println("Elasticsearch indices created successfully!")
	}
}

// pingElasticsearch tests the Elasticsearch connection
func pingElasticsearch(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", ESBaseURL, nil)
	if err != nil {
		return err
	}

	resp, err := ESClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("elasticsearch returned status %d", resp.StatusCode)
	}

	return nil
}

// createIndices creates the users and videos indices if they don't exist
func createIndices(ctx context.Context) error {
	// Create users index with its own independent timeout context
	usersCtx, usersCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer usersCancel()
	
	usersMapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"uid": map[string]interface{}{
					"type": "keyword",
				},
				"username": map[string]interface{}{
					"type":            "text",
					"analyzer":        "standard",
					"search_analyzer": "standard",
				},
				"profile_picture": map[string]interface{}{
					"type": "keyword",
				},
			},
		},
	}

	if err := createIndex(usersCtx, UsersIndex, usersMapping); err != nil {
		return fmt.Errorf("failed to create users index: %w", err)
	}

	// Create videos index with its own independent timeout context
	videosCtx, videosCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer videosCancel()
	
	videosMapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"video_id": map[string]interface{}{
					"type": "keyword",
				},
				"video_title": map[string]interface{}{
					"type":            "text",
					"analyzer":        "standard",
					"search_analyzer": "standard",
				},
				"video_description": map[string]interface{}{
					"type":            "text",
					"analyzer":        "standard",
					"search_analyzer": "standard",
				},
				"video_tags": map[string]interface{}{
					"type": "keyword",
				},
			},
		},
	}

	if err := createIndex(videosCtx, VideosIndex, videosMapping); err != nil {
		return fmt.Errorf("failed to create videos index: %w", err)
	}

	return nil
}

// createIndex creates an Elasticsearch index if it doesn't exist
func createIndex(ctx context.Context, indexName string, mapping map[string]interface{}) error {
	// Check if index exists
	checkURL := fmt.Sprintf("%s/%s", ESBaseURL, indexName)
	req, err := http.NewRequestWithContext(ctx, "HEAD", checkURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HEAD request for index %s: %w", indexName, err)
	}

	resp, err := ESClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check if index %s exists: %w", indexName, err)
	}
	resp.Body.Close()

	// Index already exists
	if resp.StatusCode == http.StatusOK {
		log.Printf("Index '%s' already exists, skipping creation", indexName)
		return nil
	}

	// Create index
	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return err
	}

	createURL := fmt.Sprintf("%s/%s", ESBaseURL, indexName)
	req, err = http.NewRequestWithContext(ctx, "PUT", createURL, bytes.NewBuffer(mappingJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = ESClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create index %s: status %d, body: %s", indexName, resp.StatusCode, string(body))
	}

	return nil
}

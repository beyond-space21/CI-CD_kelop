package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	ES "hifi/Services/Elasticsearch"
	Utils "hifi/Utils"
)

// Handle sets up the routes for search endpoints
func Handle(r chi.Router) {
	r.Get("/users/{query}", SearchUsersHandler)
	r.Get("/videos/{query}", SearchVideosHandler)
}

// SearchUsersHandler handles HTTP requests for searching users
func SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := chi.URLParam(r, "query")
	if query == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Query parameter is required")
		return
	}

	// Parse limit from query params (optional, default 10, max 100)
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= 100 {
				limit = l
			} else if l > 100 {
				limit = 100
			}
		}
	}

	// Perform search
	results, err := SearchUsers(ctx, query, limit)
	if err != nil {
		log.Printf("SearchUsersHandler: failed to search users: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to search users")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"users": results,
		"query": query,
		"limit": limit,
		"count": len(results),
	})
}

// SearchVideosHandler handles HTTP requests for searching videos
func SearchVideosHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := chi.URLParam(r, "query")
	if query == "" {
		Utils.SendErrorResponse(w, http.StatusBadRequest, "Query parameter is required")
		return
	}

	// Parse limit from query params (optional, default 10, max 100)
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= 100 {
				limit = l
			} else if l > 100 {
				limit = 100
			}
		}
	}

	// Perform search
	results, err := SearchVideos(ctx, query, limit)
	if err != nil {
		log.Printf("SearchVideosHandler: failed to search videos: %v", err)
		Utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to search videos")
		return
	}

	Utils.SendSuccessResponse(w, map[string]interface{}{
		"videos": results,
		"query":  query,
		"limit":  limit,
		"count":  len(results),
	})
}

// IndexUser indexes a user document in Elasticsearch
func IndexUser(ctx context.Context, uid, username, profilePicture string) error {
	if !ES.IsESEnabled() {
		return nil // Silently skip if Elasticsearch is not enabled
	}

	doc := map[string]interface{}{
		"uid":             uid,
		"username":        username,
		"profile_picture": profilePicture,
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal user document: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", ES.GetESBaseURL(), ES.GetUsersIndex(), uid)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(docJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ES.GetESClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to index user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to index user: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// IndexVideo indexes a video document in Elasticsearch
func IndexVideo(ctx context.Context, videoID, title, description string, tags []string, userUsername string) error {
	if !ES.IsESEnabled() {
		return nil // Silently skip if Elasticsearch is not enabled
	}

	doc := map[string]interface{}{
		"video_id":          videoID,
		"video_title":       title,
		"video_description": description,
		"video_tags":        tags,
		"user_username":     userUsername,
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal video document: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", ES.GetESBaseURL(), ES.GetVideosIndex(), videoID)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(docJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ES.GetESClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to index video: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to index video: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteUser deletes a user document from Elasticsearch
func DeleteUser(ctx context.Context, uid string) error {
	if !ES.IsESEnabled() {
		return nil // Silently skip if Elasticsearch is not enabled
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", ES.GetESBaseURL(), ES.GetUsersIndex(), uid)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ES.GetESClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	defer resp.Body.Close()

	// 404 is acceptable (document doesn't exist)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteVideo deletes a video document from Elasticsearch
func DeleteVideo(ctx context.Context, videoID string) error {
	if !ES.IsESEnabled() {
		return nil // Silently skip if Elasticsearch is not enabled
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", ES.GetESBaseURL(), ES.GetVideosIndex(), videoID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ES.GetESClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}
	defer resp.Body.Close()

	// 404 is acceptable (document doesn't exist)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete video: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SearchUsers searches for users by username
func SearchUsers(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	if !ES.IsESEnabled() {
		return nil, fmt.Errorf("elasticsearch is not enabled")
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"username^2", "profile_picture"},
			},
		},
		"size": limit,
	}

	searchJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", ES.GetESBaseURL(), ES.GetUsersIndex())
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(searchJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ES.GetESClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// Handle 503 Service Unavailable (shards not ready)
		if resp.StatusCode == http.StatusServiceUnavailable {
			// Parse error to check if it's a shard availability issue
			var esError map[string]interface{}
			if json.Unmarshal(body, &esError) == nil {
				if errorObj, ok := esError["error"].(map[string]interface{}); ok {
					if errorType, ok := errorObj["type"].(string); ok && errorType == "search_phase_execution_exception" {
						// Index exists but shards aren't ready - return empty results instead of error
						log.Printf("SearchUsers: Elasticsearch shards not ready, returning empty results")
						return []map[string]interface{}{}, nil
					}
				}
			}
		}

		return nil, fmt.Errorf("failed to search users: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	hits, ok := result["hits"].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	hitsArray, ok := hits["hits"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	results := make([]map[string]interface{}, 0, len(hitsArray))
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		results = append(results, source)
	}

	return results, nil
}

// SearchVideos searches for videos by title, tags, or description
func SearchVideos(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	if !ES.IsESEnabled() {
		return nil, fmt.Errorf("elasticsearch is not enabled")
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Split query into terms for tag matching
	queryTerms := strings.Fields(strings.ToLower(query))

	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"multi_match": map[string]interface{}{
							"query":  query,
							"fields": []string{"video_title^3", "video_description^2"},
						},
					},
					{
						"terms": map[string]interface{}{
							"video_tags": queryTerms,
						},
					},
				},
				"minimum_should_match": 1,
			},
		},
		"size": limit,
	}

	searchJSON, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", ES.GetESBaseURL(), ES.GetVideosIndex())
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(searchJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ES.GetESClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// Handle 503 Service Unavailable (shards not ready)
		if resp.StatusCode == http.StatusServiceUnavailable {
			// Parse error to check if it's a shard availability issue
			var esError map[string]interface{}
			if json.Unmarshal(body, &esError) == nil {
				if errorObj, ok := esError["error"].(map[string]interface{}); ok {
					if errorType, ok := errorObj["type"].(string); ok && errorType == "search_phase_execution_exception" {
						// Index exists but shards aren't ready - return empty results instead of error
						log.Printf("SearchVideos: Elasticsearch shards not ready, returning empty results")
						return []map[string]interface{}{}, nil
					}
				}
			}
		}

		return nil, fmt.Errorf("failed to search videos: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	hits, ok := result["hits"].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	hitsArray, ok := hits["hits"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	results := make([]map[string]interface{}, 0, len(hitsArray))
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}

		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		results = append(results, source)
	}

	return results, nil
}

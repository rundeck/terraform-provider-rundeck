// The MIT License (MIT)

// Copyright (c) 2015 Martin Atkins

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package rundeck

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rundeck/go-rundeck/rundeck"
	"github.com/rundeck/go-rundeck/rundeck/auth"
)

// =============================================================================
// JOB API - JSON
// =============================================================================
//
// This file contains structs and functions for interacting with the Rundeck
// Job API using JSON format.
//
// The provider uses JSON for all API interactions with Rundeck v5.0.0+ (API v46+).
//
// For job import/export operations, see resource_job_framework.go
// =============================================================================

// JobJSON represents the Rundeck Job JSON format returned by the API.
//
// This struct is used by:
// - GetJobJSON(): Read job details from API (GET /job/{id})
// - Framework resource (resource_job_framework.go): Read operations
//
// Key features:
// - Pure JSON format (no legacy XML tags)
// - Maps complex nested structures as map[string]interface{} or []map[string]interface{}
// - Used with custom HTTP client that explicitly requests application/json
// - Matches Rundeck API v46+ JSON response format
type JobJSON struct {
	ID                     string                   `json:"id"`
	Name                   string                   `json:"name"`
	Group                  string                   `json:"group,omitempty"`
	Project                string                   `json:"project"`
	Description            string                   `json:"description"`
	ExecutionEnabled       bool                     `json:"executionEnabled"`
	ScheduleEnabled        bool                     `json:"scheduleEnabled"`
	LogLevel               string                   `json:"loglevel,omitempty"`
	AllowConcurrentExec    bool                     `json:"multipleExecutions,omitempty"`
	NodeFilterEditable     bool                     `json:"nodeFilterEditable"`
	NodesSelectedByDefault bool                     `json:"nodesSelectedByDefault"`
	DefaultTab             string                   `json:"defaultTab,omitempty"`
	Timeout                string                   `json:"timeout,omitempty"`
	Retry                  map[string]string        `json:"retry,omitempty"`
	Options                []map[string]interface{} `json:"options,omitempty"`
	Sequence               map[string]interface{}   `json:"sequence,omitempty"`
	Notification           map[string]interface{}   `json:"notification,omitempty"`
	NodeFilters            map[string]string        `json:"nodefilters,omitempty"`
	Dispatch               map[string]interface{}   `json:"dispatch,omitempty"`
	Schedule               map[string]interface{}   `json:"schedule,omitempty"`
	Orchestrator           map[string]interface{}   `json:"orchestrator,omitempty"`
	Plugins                map[string]interface{}   `json:"plugins,omitempty"`
	LogLimit               *string                  `json:"loglimit,omitempty"`
	LogLimitAction         *string                  `json:"loglimitAction,omitempty"`
	LogLimitStatus         *string                  `json:"loglimitStatus,omitempty"`
}

// GetJobJSON returns the job details from the Rundeck API.
//
// This function uses a custom HTTP client to explicitly request JSON format
// (application/json header) to ensure consistent API responses.
//
// Returns:
// - *JobJSON: The job details
// - error: NotFoundError if job doesn't exist, or other errors
func GetJobJSON(c *rundeck.BaseClient, id string) (*JobJSON, error) {
	// Use custom HTTP request to get JSON format
	// The SDK's JobGet method doesn't work properly with JSON at v56
	url := c.BaseURI + "/job/" + id

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Accept header to request JSON response
	req.Header.Set("Accept", "application/json")

	// Add auth token header - extract from TokenAuthorizer
	if c.Authorizer != nil {
		// The Authorizer is a TokenAuthorizer with a Token field
		if tokenAuth, ok := c.Authorizer.(*auth.TokenAuthorizer); ok {
			req.Header.Set("X-Rundeck-Auth-Token", tokenAuth.Token)
		}
	}

	// Make the request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, &NotFoundError{}
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error getting job: status %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON response - API returns an array
	var jobs []JobJSON
	if err := json.Unmarshal(respBytes, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse job JSON: %w", err)
	}

	if len(jobs) == 0 {
		return nil, &NotFoundError{}
	}

	return &jobs[0], nil
}

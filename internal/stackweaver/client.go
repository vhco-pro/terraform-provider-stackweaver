// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

// Package stackweaver is the native API client for Stackweaver-only resources
// that have no terraform-provider-tfe equivalent (the Ansible surface, runners,
// VCS connections, ...). It mirrors go-tfe's house style: a Client holding an
// *http.Client + base URL + token, with one file per resource family exposing
// typed List/Create/Read/Update/Delete methods.
//
// The native API envelope is mixed — most Ansible resources are JSON:API, a few
// endpoints are plain JSON — so the client carries BOTH a JSON:API codec and a
// plain-JSON codec and each service method declares which it uses.
package stackweaver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// apiPrefix is the versioned API root every native endpoint hangs off. Service
// method paths are expressed relative to it (e.g. "/ansible/playbooks/:id").
const apiPrefix = "/api/v2"

// ErrNotImplemented is returned by scaffold service methods that have not been
// wired up yet. Native waves replace these with real implementations.
var ErrNotImplemented = errors.New("stackweaver: not implemented")

// ErrNotFound is the sentinel returned by do when the API answers 404. Resources
// detect it (via errors.Is) to distinguish a deleted/absent resource from a real
// error, so Read can drop the resource from state instead of failing the plan.
var ErrNotFound = errors.New("stackweaver: resource not found")

// APIError is the typed error returned for a non-2xx (and non-404) response. It
// carries the HTTP status plus the decoded JSON:API error envelope when present.
type APIError struct {
	StatusCode int
	Title      string
	Detail     string
	// Body is the raw response body, retained for diagnostics when the payload
	// is not a recognizable JSON:API error envelope.
	Body string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	switch {
	case e.Detail != "":
		return fmt.Sprintf("stackweaver: API error %d: %s", e.StatusCode, e.Detail)
	case e.Title != "":
		return fmt.Sprintf("stackweaver: API error %d: %s", e.StatusCode, e.Title)
	case e.Body != "":
		return fmt.Sprintf("stackweaver: API error %d: %s", e.StatusCode, e.Body)
	default:
		return fmt.Sprintf("stackweaver: API error %d", e.StatusCode)
	}
}

// Config configures a native Stackweaver Client.
type Config struct {
	// BaseURL is the Stackweaver API base, e.g. "https://app.stackweaver.io".
	BaseURL string
	// Token is the bearer/API token used to authenticate requests.
	Token string
	// HTTPClient is optional; a default client is used when nil.
	HTTPClient *http.Client
}

// Client is the native Stackweaver API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client

	// codecs. Each service method selects the codec its endpoint uses.
	jsonapi *jsonAPICodec
	plain   *plainJSONCodec

	// Services. One field per resource family (added over the native waves).
	AnsiblePlaybooks *AnsiblePlaybooksService
}

// NewClient constructs a native Stackweaver Client from cfg.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("stackweaver: BaseURL is required")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("stackweaver: Token is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	c := &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		token:      cfg.Token,
		httpClient: httpClient,
		jsonapi:    &jsonAPICodec{},
		plain:      &plainJSONCodec{},
	}

	c.AnsiblePlaybooks = &AnsiblePlaybooksService{client: c}

	return c, nil
}

// do executes an authenticated request against baseURL+apiPrefix+path. body is an
// already-encoded request body (nil for GET/DELETE). On success it returns the
// raw response body; callers decode it with the codec their endpoint uses. Error
// mapping: a 404 becomes ErrNotFound (so resources can detect deletion), any
// other non-2xx becomes an *APIError carrying the decoded error envelope.
func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := c.baseURL + apiPrefix + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("stackweaver: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.api+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/vnd.api+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stackweaver: execute %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stackweaver: read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// newAPIError decodes the JSON:API error envelope ({"errors":[{...}]}) from body
// when possible, falling back to the raw body.
func newAPIError(status int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: status, Body: strings.TrimSpace(string(body))}

	var env struct {
		Errors []struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &env); err == nil && len(env.Errors) > 0 {
		apiErr.Title = env.Errors[0].Title
		apiErr.Detail = env.Errors[0].Detail
	}

	return apiErr
}

// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

// Package stackweaver is the native API client for Stackweaver-only resources
// that have no terraform-provider-tfe equivalent (the Ansible surface, runners,
// VCS connections, ...). It mirrors go-tfe's house style: a Client holding an
// *http.Client + base URL + token, with one file per resource family exposing
// typed List/Create/Read/Update/Delete methods.
//
// This is the bootstrap scaffold (plan.md §4 / tasks.md T0 step 4). It compiles
// and defines the shape every native service follows; the code-heavy native
// waves (T-native-*) fill in real request/response logic. The native API
// envelope is mixed — most Ansible resources are JSON:API, a few endpoints are
// plain JSON — so the client carries BOTH a JSON:API codec and a plain-JSON
// codec and each service method declares which it uses.
package stackweaver

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

// ErrNotImplemented is returned by scaffold service methods that have not been
// wired up yet. Native waves replace these with real implementations.
var ErrNotImplemented = errors.New("stackweaver: not implemented")

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

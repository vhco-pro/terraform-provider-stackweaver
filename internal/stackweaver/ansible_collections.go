// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import (
	"context"
	"net/http"
)

// AnsibleCollectionsService lists the Ansible Galaxy collections pre-installed
// on the Stackweaver runner image. It backs the stackweaver_ansible_collections
// data source.
//
// Self-contained service: the data source constructs it via
// NewAnsibleCollectionsService(cc.Stackweaver). The endpoint is a JSON:API
// collection ({"data":[{"type":"ansible-collections",...}]}). Wire contract
// (path relative to /api/v2):
//
//	ListPreInstalled: GET /ansible/collections/pre-installed
//
// The Galaxy search endpoint is a backend placeholder (empty results), so this
// service models pre-installed only.
type AnsibleCollectionsService struct {
	client *Client
}

// NewAnsibleCollectionsService constructs the service against c.
func NewAnsibleCollectionsService(c *Client) *AnsibleCollectionsService {
	return &AnsibleCollectionsService{client: c}
}

// AnsibleCollection is one pre-installed Galaxy collection.
type AnsibleCollection struct {
	Name        string
	Namespace   string
	Version     string
	Description string
	Source      string
}

// collectionResource mirrors the JSON:API resource object. The item id is the
// collection name; attributes carry the (snake-free) plain keys the handler emits.
type collectionResource struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name        string `json:"name"`
		Namespace   string `json:"namespace"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Source      string `json:"source"`
	} `json:"attributes"`
}

// ListPreInstalled returns the collections pre-installed on the runner image.
func (s *AnsibleCollectionsService) ListPreInstalled(ctx context.Context) ([]*AnsibleCollection, error) {
	body, err := s.client.do(ctx, http.MethodGet, "/ansible/collections/pre-installed", nil)
	if err != nil {
		return nil, err
	}

	var resources []collectionResource
	if err := s.client.jsonapi.Unmarshal(body, &resources); err != nil {
		return nil, err
	}

	collections := make([]*AnsibleCollection, len(resources))
	for i := range resources {
		collections[i] = &AnsibleCollection{
			Name:        resources[i].Attributes.Name,
			Namespace:   resources[i].Attributes.Namespace,
			Version:     resources[i].Attributes.Version,
			Description: resources[i].Attributes.Description,
			Source:      resources[i].Attributes.Source,
		}
	}
	return collections, nil
}

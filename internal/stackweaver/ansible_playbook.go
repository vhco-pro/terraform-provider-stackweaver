// Copyright (c) VH & Co BV.
// SPDX-License-Identifier: MPL-2.0

package stackweaver

import "context"

// AnsiblePlaybooksService is the reference native service — the template every
// other native resource family follows (plan.md §4). ansible_playbook is a
// JSON:API resource, so its methods use the JSON:API codec. Bodies are stubs
// until Native Wave A (tasks.md); the method signatures define the contract.
type AnsiblePlaybooksService struct {
	client *Client
}

// AnsiblePlaybook is the native representation of an Ansible playbook.
type AnsiblePlaybook struct {
	ID          string `jsonapi:"primary,ansible-playbooks"`
	Name        string `jsonapi:"attr,name"`
	Description string `jsonapi:"attr,description,omitempty"`
	ProjectID   string `jsonapi:"attr,project-id,omitempty"`
	Path        string `jsonapi:"attr,path,omitempty"`
}

// AnsiblePlaybookListOptions are the filter/pagination options for List.
type AnsiblePlaybookListOptions struct {
	ProjectID string
	PageSize  int
	PageNum   int
}

// AnsiblePlaybookCreateOptions are the fields accepted when creating a playbook.
type AnsiblePlaybookCreateOptions struct {
	Name        string
	Description string
	ProjectID   string
	Path        string
}

// AnsiblePlaybookUpdateOptions are the fields accepted when updating a playbook.
type AnsiblePlaybookUpdateOptions struct {
	Name        *string
	Description *string
	Path        *string
}

// List returns the playbooks matching options.
func (s *AnsiblePlaybooksService) List(ctx context.Context, options AnsiblePlaybookListOptions) ([]*AnsiblePlaybook, error) {
	return nil, ErrNotImplemented
}

// Create creates a new playbook.
func (s *AnsiblePlaybooksService) Create(ctx context.Context, options AnsiblePlaybookCreateOptions) (*AnsiblePlaybook, error) {
	return nil, ErrNotImplemented
}

// Read returns the playbook identified by id.
func (s *AnsiblePlaybooksService) Read(ctx context.Context, id string) (*AnsiblePlaybook, error) {
	return nil, ErrNotImplemented
}

// Update updates the playbook identified by id.
func (s *AnsiblePlaybooksService) Update(ctx context.Context, id string, options AnsiblePlaybookUpdateOptions) (*AnsiblePlaybook, error) {
	return nil, ErrNotImplemented
}

// Delete removes the playbook identified by id.
func (s *AnsiblePlaybooksService) Delete(ctx context.Context, id string) error {
	return ErrNotImplemented
}

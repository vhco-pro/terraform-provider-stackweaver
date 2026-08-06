# Copyright (c) VH & Co BV. SPDX-License-Identifier: MPL-2.0
NAME := terraform-provider-stackweaver

default: build

build:
	go build -o $(NAME) .

install: build
	go install .

test:
	go test ./internal/stackweaver/ -count=1
	go test ./internal/provider/ -run 'DualPrefix|KeptSurface' -count=1

# Acceptance tests need a live Stackweaver stack + TFE_TOKEN (run via dev_overrides).
testacc:
	TF_ACC=1 go test ./internal/provider/ -v -count=1 -timeout 30m

fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint:
	golangci-lint run

.PHONY: default build install test testacc fmt vet lint

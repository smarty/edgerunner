#!/usr/bin/make -f

test: fmt
	go test -v -count=1 -timeout=5s -short -race -covermode=atomic # ./...

fmt:
	go mod tidy && go fmt ./...

compile:
	go build ./...

build: test compile

.PHONY: test fmt compile build

#!/usr/bin/make -f

test: fmt
	go test -v -count=1 -short=false -race -covermode=atomic # ./...

fmt:
	go fmt ./...

compile:
	go build ./...

build: test compile

.PHONY: test fmt compile build

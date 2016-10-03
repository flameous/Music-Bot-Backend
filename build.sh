#!/usr/bin/env bash

export GOPATH=`pwd`

env GOOS=windows GOARCH=amd64 go build src/main.go
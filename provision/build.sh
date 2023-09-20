#!/usr/bin/env bash

go build -i -v -ldflags="-X main.version=$(git describe --always --long)" .


#!/bin/bash
mkdir -p output/bin
go build -o output/bin/httpgen ./cmd/httpgen
output/bin/httpgen -handler handler.gen.go -router router.gen.go -output example/httpgen example/example.thrift

go build -o output/bin/example-server ./example/server
output/bin/example-server
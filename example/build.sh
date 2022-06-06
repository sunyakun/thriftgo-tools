#!/bin/bash
set -e
set -x

cd `dirname "$0"` && cd ..

function build_and_gen() {
   # build binary executable
    mkdir -p output/bin
    rm -f output/bin/thriftgo-http-plugin
    go build -o output/bin/httpgen ./cmd/httpgen
    go build -o output/bin/combine ./cmd/combine

    # generate code by thrift
    output/bin/httpgen -handler handler.gen.go -router router.gen.go -output example/httpgen/http_gen example/example.thrift
    output/bin/httpgen -handler handler.gen.go -router router.gen.go -output example/httpgen/http_gen example/another_example.thrift

    # combine multiple thrift files into one
    output/bin/combine -input_files example/example.thrift,example/another_example.thrift -output example/combine_service.thrift -namespace combine_service

    # generate code by combined thrift file
    output/bin/httpgen -handler handler.gen.go -router router.gen.go -output example/httpgen/http_gen -prefix github.com/sunyakun/thriftgo-tools/example/httpgen/http_gen example/combine_service.thrift

    go build -o output/bin/example-server ./example/httpgen/server
}

function serve() {
  output/bin/example-server
}

function clean() {
  rm -rf example/httpgen/http_gen/example
  rm -rf example/httpgen/http_gen/another_example
  rm -rf example/httpgen/http_gen/combine_service
  rm -f example/combine_service.thrift
}

case $1 in
    "build")
        build_and_gen
        ;;
    "clean")
        clean
        ;;
    "serve")
        serve
        ;;
    "")
        echo "Usage: ./build.sh [build|clean|serve]"
        exit 1
        ;;
esac
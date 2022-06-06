package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cloudwego/thriftgo/parser"

	thriftgo_tools "github.com/sunyakun/thriftgo-tools"
)

func empty(val, name string) error {
	if val == "" {
		return fmt.Errorf("%s is empty", name)
	}
	return nil
}

func main() {
	inputFilesStr := flag.String("input_files", "", "input files")
	outputPath := flag.String("output", "", "output file path")
	namespace := flag.String("namespace", "", "thrift namespace")
	flag.Parse()

	if err := empty(*inputFilesStr, "input_files"); err != nil {
		panic(err)
	}
	if err := empty(*outputPath, "output"); err != nil {
		panic(err)
	}
	if err := empty(*namespace, "namespace"); err != nil {
		panic(err)
	}

	inputFiles := strings.Split(*inputFilesStr, ",")

	asts := make([]*parser.Thrift, 0, len(inputFiles))
	for _, file := range inputFiles {
		ast, err := parser.ParseFile(file, nil, false)
		if err != nil {
			panic(err)
		}
		asts = append(asts, ast)
	}

	content, err := thriftgo_tools.Combine(asts, *outputPath, *namespace)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(*outputPath, content, 0644)
	if err != nil {
		panic(err)
	}
}

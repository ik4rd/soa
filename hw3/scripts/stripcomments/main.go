package main

import (
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
)

func main() {
	for _, path := range os.Args[1:] {
		files := token.NewFileSet()
		node, err := parser.ParseFile(files, path, nil, 0)
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}

		if err := format.Node(file, files, node); err != nil {
			log.Fatal(err)
		}

		if err := file.Close(); err != nil {
			log.Fatal(err)
		}
	}
}

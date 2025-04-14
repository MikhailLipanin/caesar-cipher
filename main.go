package main

import (
	"fmt"
	"log"
	"os"
	"plugin"
	"strconv"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run main.go <enc/dec> <file> <keyword>")
		return
	}
	mode := os.Args[1]
	filePath := os.Args[2]
	keyword := os.Args[3]
	shiftArg := os.Args[4]

	shift, err := strconv.Atoi(shiftArg)
	if err != nil {
		log.Fatal(err)
	}

	p, err := plugin.Open(".build/caesar_keyword.so")
	if err != nil {
		log.Fatal(err)
	}

	var processFunc func(string, string, int) (string, error)
	switch mode {
	case "enc":
		sym, err := p.Lookup("Encrypt")
		if err != nil {
			log.Fatal(err)
		}
		processFunc = sym.(func(string, string, int) (string, error))
	case "dec":
		sym, err := p.Lookup("Decrypt")
		if err != nil {
			log.Fatal(err)
		}
		processFunc = sym.(func(string, string, int) (string, error))
	default:
		log.Fatal("Invalid mode")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	processedText, err := processFunc(string(data), keyword, shift)
	if err != nil {
		log.Fatal(err)
	}

	resultFilePath := filePath + "." + mode
	err = os.WriteFile(resultFilePath, []byte(processedText), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Processed file saved as %s\n", resultFilePath)
}

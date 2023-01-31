package main

import (
	"flag"
	"fmt"
	"os"
)

func runFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open %s: %s\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()
	c := NewContext()
	c.LoadBuiltins()
	_, err = c.Eval(file, filePath)
	if err != nil {
		fmt.Println(err)
	}
}

func checkFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open %s: %s\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()
	c := NewTypecheckContext()
	// c.LoadBuiltins()
	_, err = c.Typecheck(file, filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	colorPrintln(ColorGreen, "No type errors found!")
}

func main() {
	check := flag.Bool("check", false, "only typecheck")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		// TODO: repl
		fmt.Println("TODO: repl")
		return
	}
	if *check {
		checkFile(args[0])
		return
	}
	runFile(args[0])
}

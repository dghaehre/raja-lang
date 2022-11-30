package main

import (
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
	_, err = c.Eval(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// if v != nil {
	// 	fmt.Println(v)
	// }
}

func main() {
	if len(os.Args) == 1 {
		// TODO: repl
		os.Exit(0)
	}
	if len(os.Args) == 2 {
		runFile(os.Args[1])
		os.Exit(0)
	}
	fmt.Println("only one argument allowed for now")
}

package main

import (
	"dghaehre/raja/typecheck"
	"dghaehre/raja/eval"
	"flag"
	"fmt"
	color "github.com/dghaehre/termcolor"
	"os"
)

func usage() string {
	header := fmt.Sprintf("%s, the programming language\n\n", color.Str(color.Blue, "Raja"))
	usage := fmt.Sprintf(`%s:
    raja [OPTIONS] [FILE]

If no FILE is given, a repl is opened.

%s:
    --check       Check given file for type errors and similar.
                  It will not run the file.

    -h, --help    Show this message
    `, color.Str(color.Yellow, "USAGE"), color.Str(color.Yellow, "OPTIONS"))

	return header + usage
}

func runFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open %s: %s\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()
	c := eval.NewContext()
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
	c := typecheck.NewTypecheckContext()
	c.LoadBuiltins()
	_, err = c.Typecheck(file, filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	color.Println(color.Green, "No type errors found!")
}

func main() {
	check := flag.Bool("check", false, "Typecheck")
	flag.Usage = func() {
		fmt.Println(usage())
	}
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

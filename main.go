package main

import (
	"dghaehre/raja/codegen"
	"dghaehre/raja/eval"
	"dghaehre/raja/typecheck"
	"flag"
	"fmt"
	"os"

	color "github.com/dghaehre/termcolor"
)

func usage() string {
	header := fmt.Sprintf("%s, the programming language\n\n", color.Str(color.Blue, "Raja"))
	usage := fmt.Sprintf(`%s:
    raja [OPTIONS] [FILE]

If no FILE is given, a repl is opened.

%s:
    --check       Check given file for type errors and similar.
                  It will not run the file.

    --build       Check given file for type errors and similar, and then build binary.
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
	c.LoadLibs()
	_, err = c.Typecheck(file, filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	color.Println(color.Green, "If it compiles it works")
}

func buildFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open %s: %s\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()
	c := typecheck.NewTypecheckContext()
	c.LoadBuiltins()
	c.LoadLibs()
	typed, err := c.Typecheck(file, filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	output := "test-bin"
	err = codegen.Build(typed, "/home/dghaehre/projects/personal/raja/" + output)
	if err != nil {
		fmt.Println(err)
		return
	}
	head := color.Str(color.Green, "Ready to ship!\n\n")
	filename := color.Str(color.Blue, output + "\n")
	fmt.Println(head + filename)
}

func main() {
	check := flag.Bool("check", false, "Typecheck")
	build := flag.Bool("build", false, "Build binary")
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
	if *build {
		buildFile(args[0])
		return
	}
	runFile(args[0])
}

package codegen

import (
	"dghaehre/raja/typecheck"
	"dghaehre/raja/util"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Env struct {
	imports []string
}

func NewEnv() *Env {
	return &Env{}
}

// TODO: check if it exists already
func (e *Env) addImport(path string) {
	if !util.Exists(e.imports, path) {
		e.imports = append(e.imports, path)
	}
}

func (e *Env) GenerateHeader() string {
	main := `package main
`
	if len(e.imports) == 0 {
		return main
	}
	if len(e.imports) == 1 {
		return main + `import "` + e.imports[0] + `"`
	}
	return main + `import (
  ` + strings.Join(e.imports, "\n") + `
  )`
}

// I need payload in TypedAstNode to be able to do anything useful!
func (e *Env) GenerateBody(tast typecheck.TypedAstNode) (string, error) {
	e.addImport("fmt")

	return `
	func main() {
		fmt.Println("heyja")
	}
	`, nil
}

func (e *Env) Generate(tast typecheck.TypedAstNode) (string, error) {
	body, err := e.GenerateBody(tast)
	if err != nil {
		return "", err
	}

	// Do this after body generation
	header := e.GenerateHeader()

	return header + body, nil
}

func Build(tast typecheck.TypedAstNode, filePath string) error {
	env := NewEnv()
	s, err := env.Generate(tast)
	if err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp("", "raja-build-*.go")
	if err != nil {
		return err
	}
	_, err = io.WriteString(tmpFile, s)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-o", filePath, tmpFile.Name())
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(tmpFile.Name())
		fmt.Println(string(out))
		return err
	}
	return nil
}

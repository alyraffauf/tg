//go:build ignore

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	data, err := os.ReadFile("cborgen/main.go")
	if err != nil {
		panic(err)
	}
	names := regexp.MustCompile(`lex\.(\w+)\{\}`).FindAllStringSubmatch(string(data), -1)
	var output strings.Builder
	output.WriteString("package tangledlex\nimport \"io\"\n")
	for _, name := range names {
		fmt.Fprintf(&output, "func(t *%s)MarshalCBOR(io.Writer)error{return nil}\nfunc(t *%s)UnmarshalCBOR(io.Reader)error{return nil}\n", name[1], name[1])
	}
	if err := os.WriteFile("cbor_gen.go", []byte(output.String()), 0600); err != nil {
		panic(err)
	}
}

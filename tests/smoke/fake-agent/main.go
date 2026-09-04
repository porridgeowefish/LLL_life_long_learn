// Command fake-agent records its invocation for the Windows wrapper smoke test.
package main

import (
	"encoding/json"
	"os"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	output := os.Getenv("LLL_SMOKE_OUTPUT")
	if output == "" {
		panic("LLL_SMOKE_OUTPUT is required")
	}
	record := struct {
		Args []string `json:"args"`
		CWD  string   `json:"cwd"`
	}{Args: os.Args[1:], CWD: cwd}
	data, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(output, data, 0o644); err != nil {
		panic(err)
	}
}

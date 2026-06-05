// genpng renders lifecycle.svg to lifecycle.png (text, fonts, full fidelity).
// Run from repo root: go run ./cmd/gui/assets/diagrams/genpng
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	dir, err := filepath.Abs(filepath.Join("cmd", "gui", "assets", "diagrams"))
	if err != nil {
		fatal(err)
	}
	svg := filepath.Join(dir, "lifecycle.svg")
	png := filepath.Join(dir, "lifecycle.png")

	npx, err := exec.LookPath("npx")
	if err != nil {
		fatal(fmt.Errorf("npx not found (install Node.js): %w", err))
	}

	cmd := exec.Command(npx, "--yes", "@resvg/resvg-js-cli", "--fit-width", "2406", svg, png)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", png)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

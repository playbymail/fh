package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/playbymail/fh/internal/engine/rng"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update various things",
}

var goldenCmd = &cobra.Command{
	Use:   "golden",
	Short: "Update golden test files",
}

var rngCmd = &cobra.Command{
	Use:   "rng",
	Short: "Update RNG golden files",
	Run: func(cmd *cobra.Command, args []string) {
		// Update algorithmm.golden
		numbers := rng.GenerateGoldenUint64(0xDEADBEEF, 100)
		var b bytes.Buffer
		for _, n := range numbers {
			b.WriteString(fmt.Sprintf("%d\n", n))
		}
		goldenFile := filepath.Join("internal", "engine", "rng", "testdata", "algorithmm.golden")
		if err := os.WriteFile(goldenFile, b.Bytes(), 0644); err != nil {
			fmt.Printf("failed to write %s: %v\n", goldenFile, err)
			os.Exit(1)
		}

		// Update algorithmm_range0to7.golden
		numbersInt := rng.GenerateGoldenIntn(0xDEADBEEF, 8, 1024)
		b.Reset()
		for _, n := range numbersInt {
			b.WriteString(fmt.Sprintf("%d\n", n))
		}
		goldenFile2 := filepath.Join("internal", "engine", "rng", "testdata", "algorithmm_range0to7.golden")
		if err := os.WriteFile(goldenFile2, b.Bytes(), 0644); err != nil {
			fmt.Printf("failed to write %s: %v\n", goldenFile2, err)
			os.Exit(1)
		}

		fmt.Println("Updated RNG golden files")
	},
}



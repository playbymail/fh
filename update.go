package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v4"
	"github.com/playbymail/fh/internal/engine/rng"
)

// newUpdateCmd returns the "update" command and its subcommands.
func newUpdateCmd(parent *ff.FlagSet) *ff.Command {
	updateGoldenRngCmd := &ff.Command{
		Name:      "rng",
		Usage:     "fh update golden rng",
		ShortHelp: "Update RNG golden files",
		Flags:     ff.NewFlagSet("rng").SetParent(parent),
		Exec: func(ctx context.Context, args []string) error {
			// Update algorithmm.golden
			numbers := rng.GenerateGoldenUint64(0xDEADBEEF, 100)
			var b bytes.Buffer
			for _, n := range numbers {
				b.WriteString(fmt.Sprintf("%d\n", n))
			}

			goldenRoot := filepath.Join("internal", "engine", "rng", "testdata")
			goldenFile := filepath.Join(goldenRoot, "algorithmm.golden")
			if err := os.WriteFile(goldenFile, b.Bytes(), 0644); err != nil {
				return fmt.Errorf("write %s: %w", goldenFile, err)
			}

			// Update algorithmm_range0to7.golden
			numbersInt := rng.GenerateGoldenIntn(0xDEADBEEF, 8, 1024)
			b.Reset()
			for _, n := range numbersInt {
				b.WriteString(fmt.Sprintf("%d\n", n))
			}
			goldenFile2 := filepath.Join(goldenRoot, "algorithmm_range0to7.golden")
			if err := os.WriteFile(goldenFile2, b.Bytes(), 0644); err != nil {
				return fmt.Errorf("write %s: %w", goldenFile2, err)
			}

			fmt.Println("Updated RNG golden files")
			return nil
		},
	}

	updateGoldenCmd := &ff.Command{
		Name:        "golden",
		Usage:       "fh update golden <subcommand>",
		ShortHelp:   "Update golden test files",
		Flags:       ff.NewFlagSet("golden").SetParent(parent),
		Subcommands: []*ff.Command{updateGoldenRngCmd},
	}

	return &ff.Command{
		Name:        "update",
		Usage:       "fh update <subcommand>",
		ShortHelp:   "Update various things",
		Flags:       ff.NewFlagSet("update").SetParent(parent),
		Subcommands: []*ff.Command{updateGoldenCmd},
	}
}

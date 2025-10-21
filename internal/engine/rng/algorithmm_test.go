package rng

import (
	"bufio"
	"os"
	"strconv"
	"testing"
)

func TestAlgorithmM(t *testing.T) {
	rng := NewAlgorithmM(0xDEADBEEF)

	var numbers []uint64
	for i := 0; i < 100; i++ {
		numbers = append(numbers, rng.Uint64())
	}

	goldenFile := "testdata/algorithmm.golden"
	file, err := os.Open(goldenFile)
	if err != nil {
		t.Fatalf("failed to open golden file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var expected []uint64
	for scanner.Scan() {
		line := scanner.Text()
		num, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse number: %v", err)
		}
		expected = append(expected, num)
	}

	if len(numbers) != len(expected) {
		t.Fatalf("length mismatch: got %d, expected %d", len(numbers), len(expected))
	}

	for i, num := range numbers {
		if num != expected[i] {
			t.Errorf("mismatch at %d: got %d, expected %d", i, num, expected[i])
		}
	}
}

func TestAlgorithmMRange0to7(t *testing.T) {
	rng := NewAlgorithmM(0xDEADBEEF)

	var numbers []int
	for i := 0; i < 1024; i++ {
		numbers = append(numbers, rng.Intn(8))
	}

	goldenFile := "testdata/algorithmm_range0to7.golden"
	file, err := os.Open(goldenFile)
	if err != nil {
		t.Fatalf("failed to open golden file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var expected []int
	for scanner.Scan() {
		line := scanner.Text()
		num, err := strconv.Atoi(line)
		if err != nil {
			t.Fatalf("failed to parse number: %v", err)
		}
		expected = append(expected, num)
	}

	if len(numbers) != len(expected) {
		t.Fatalf("length mismatch: got %d, expected %d", len(numbers), len(expected))
	}

	for i, num := range numbers {
		if num != expected[i] {
			t.Errorf("mismatch at %d: got %d, expected %d", i, num, expected[i])
		}
	}
}

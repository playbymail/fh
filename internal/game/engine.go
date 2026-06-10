package game

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Port of engine.c plus small file helpers standing in for C stdio.

// cfile wraps an os.File with a buffered reader so the parser and log
// code can do C-style fgets line reading.
type cfile struct {
	f    *os.File
	r    *bufio.Reader
	name string
}

// fopen_r opens a file for reading, returning nil if it cannot be opened
// (matching C fopen returning NULL).
func fopen_r(name string) *cfile {
	f, err := os.Open(name)
	if err != nil {
		return nil
	}
	return &cfile{f: f, r: bufio.NewReader(f), name: name}
}

func (fp *cfile) fclose() {
	if fp != nil && fp.f != nil {
		fp.f.Close()
	}
}

// fgets reads at most n-1 bytes, stopping after a newline, like C fgets.
// It returns false at end of file when no bytes were read.
func (fp *cfile) fgets(n int) (string, bool) {
	var buf []byte
	for len(buf) < n-1 {
		c, err := fp.r.ReadByte()
		if err != nil {
			if len(buf) == 0 {
				return "", false
			}
			break
		}
		buf = append(buf, c)
		if c == '\n' {
			break
		}
	}
	return string(buf), true
}

// readln is a helper for command parsing that coerces all line-endings to
// be just '\n'. The C version reads through a 1024-byte buffer and
// truncates the result to len-1 bytes.
func readln(fp *cfile, length int) (string, bool) {
	buf, ok := fp.fgets(1024)
	if !ok {
		return "", false
	}
	// Replace the first '\r' or '\n' with '\n' and drop everything after
	// it, exactly like the C loop that coerces line endings.
	for i := 0; i < len(buf); i++ {
		if buf[i] == '\r' || buf[i] == '\n' {
			buf = buf[:i] + "\n"
			break
		}
	}
	if len(buf) > length-1 {
		buf = buf[:length-1]
	}
	return buf, true
}

/* The following routine will return a score indicating how closely two strings match.
 * If the score is exactly 10000, then the strings are identical.
 * Otherwise, the value returned is the number of character matches, allowing for accidental transpositions, insertions, and deletions.
 * Excess characters in either string will subtract from the score.
 * Thus, it's possible for a score to be negative.
 *
 * In general, if the strings are at least 7 characters each, then you can assume the strings
 * are the same if the highest score equals the length of the correct string, length-1,
 * or length-2, AND if the score of the next best match is less than the highest score.
 * A non-10000 score will never be higher than the length of the correct string. */
func agrep_score(correct_string, unknown_string string) int {
	if correct_string == unknown_string {
		return 10000
	}
	// at mimics reading the NUL terminator at the end of a C string.
	at := func(s string, i int) byte {
		if i < len(s) {
			return s[i]
		}
		return 0
	}
	score := 0
	p1, p2 := 0, 0
	for {
		c1 := at(correct_string, p1)
		p1++
		if c1 == 0 {
			score -= len(unknown_string) - p2 /* Reduce score by excess characters, if any. */
			break
		}
		c2 := at(unknown_string, p2)
		p2++
		if c2 == 0 {
			score -= len(correct_string) - p1 /* Reduce score by excess characters, if any. */
			break
		}
		if c1 == c2 {
			score++
		} else if c1 == at(unknown_string, p2) && c2 == at(correct_string, p1) {
			/* Transposed. */
			score += 2
			p1++
			p2++
		} else if c1 == at(unknown_string, p2) {
			/* Unneeded character. */
			score++
			p2++
		} else if c2 == at(correct_string, p1) {
			/* Missing character. */
			score++
			p1++
		}
	}
	return score
}

/* This routine is intended to take a long argument and return a pointer to a string that has embedded commas to make the string more readable. */
func commas(value int) string {
	var result_plus_commas [33]byte
	negative := false
	abs_value := value
	if value < 0 {
		abs_value = -value
		negative = true
	}
	temp := strconv.Itoa(abs_value)
	length := len(temp)
	i := length - 1
	j := 31
	for n := 0; n < length; n++ {
		result_plus_commas[j] = temp[i]
		j--
		i--
		if j%4 == 0 {
			result_plus_commas[j] = ','
			j--
		}
	}
	j++
	if result_plus_commas[j] == ',' {
		j++
	}
	if negative {
		j--
		result_plus_commas[j] = '-'
	}
	return string(result_plus_commas[j:32])
}

// stdin_reader is shared so consecutive prompts do not lose buffered input.
var stdin_reader = bufio.NewReader(os.Stdin)

/* Give the gamemaster a chance to abort. */
func gamemaster_abort_option() {
	fmt.Printf("*** Gamemaster safe-abort option ... type q or Q to quit: ")
	answer, err := stdin_reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return
	}
	if len(answer) > 0 && (answer[0] == 'q' || answer[0] == 'Q') {
		os.Exit(0)
	}
}

// logRandomCommand generates random numbers using the historical default
// seed value. It is used to validate the PRNG against the C engine.
func logRandomCommand(w io.Writer) int {
	// use the historical default seed value
	prngSetSeed(defaultHistoricalSeedValue)
	// then print out a nice set of random values
	for i := 0; i < 1000000; i++ {
		r := rnd(1024 * 1024)
		if i < 10 {
			fmt.Fprintf(w, "%9d %9d\n", i, r)
		} else if 1000 < i && i < 1010 {
			fmt.Fprintf(w, "%9d %9d\n", i, r)
		} else if (i % 85713) == 0 {
			fmt.Fprintf(w, "%9d %9d\n", i, r)
		}
	}
	return 0
}

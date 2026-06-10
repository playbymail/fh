package game

import (
	"fmt"
	"os"
)

// Port of log.c. The line-wrapping logic in log_char must match the C
// engine byte for byte so generated logs and reports compare equal.

var log_indentation = 0
var log_line [128]byte
var log_position = 0
var log_start_of_line = TRUE

func resetLogState() {
	log_indentation = 0
	log_line = [128]byte{}
	log_position = 0
	log_start_of_line = TRUE
}

// log_write_line writes the NUL-terminated prefix of log_line to the
// active outputs (the C code calls fputs three times).
func log_write_line() {
	n := 0
	for n < len(log_line) && log_line[n] != 0 {
		n++
	}
	line := log_line[:n]
	if log_to_file != FALSE && log_file != nil {
		log_file.Write(line)
	}
	if log_stdout != FALSE {
		os.Stdout.Write(line)
	}
	if log_summary != FALSE && summary_file != nil {
		summary_file.Write(line)
	}
}

/* The following routines will post an item to standard output and to an externally defined log file and summary file. */

func log_char(c byte) {
	if logging_disabled != FALSE {
		return
	}

	/* Check if current line is getting too long. */
	if (c == ' ' || c == '\n') && log_position > 77 {
		/* Find closest preceeding space. */
		temp_position := log_position - 1
		for log_line[temp_position] != ' ' {
			temp_position--
		}
		/* Write front of line to files. */
		temp_char := log_line[temp_position+1]
		log_line[temp_position] = '\n'
		log_line[temp_position+1] = 0
		log_write_line()
		log_line[temp_position+1] = temp_char
		/* Copy overflow word to beginning of next line. */
		log_line[log_position] = 0
		log_position = log_indentation + 2
		for i := 0; i < log_position; i++ {
			log_line[i] = ' '
		}
		// strcpy(&log_line[log_position], &log_line[temp_position+1])
		src := temp_position + 1
		dst := log_position
		for log_line[src] != 0 {
			log_line[dst] = log_line[src]
			dst++
			src++
		}
		log_line[dst] = 0
		log_position = dst
		if c == ' ' {
			log_line[log_position] = ' '
			log_position++
			return
		}
	}

	/* Check if line is being manually terminated. */
	if c == '\n' {
		/* Write current line to output. */
		log_line[log_position] = '\n'
		log_line[log_position+1] = 0
		log_write_line()
		/* Set up for next line. */
		log_position = 0
		log_indentation = 0
		log_start_of_line = TRUE
		return
	}

	/* Save this character. */
	log_line[log_position] = c
	log_position++

	if log_start_of_line != FALSE && c == ' ' {
		/* Determine number of indenting spaces for current line. */
		log_indentation++
	} else {
		log_start_of_line = FALSE
	}
}

func log_int(value int) {
	if logging_disabled == FALSE {
		log_printf("%d", value)
	}
}

func log_long(value int) {
	if logging_disabled == FALSE {
		log_printf("%d", value)
	}
}

func log_message(message_filename string) {
	/* Open message file. */
	message_file := fopen_r(message_filename)
	if message_file == nil {
		fmt.Fprintf(os.Stderr, "\n\tWARNING! log_message: cannot open message file '%s'!\n\n", message_filename)
		return
	}
	/* Copy message to log file. */
	for {
		message_line, ok := readln(message_file, 256)
		if !ok {
			break
		}
		if log_file != nil {
			fmt.Fprint(log_file, message_line)
		}
	}
	message_file.fclose()
}

func log_printf(format string, args ...interface{}) {
	if logging_disabled == FALSE {
		buffer := fmt.Sprintf(format, args...)
		for i := 0; i < len(buffer); i++ {
			log_char(buffer[i])
		}
	}
}

func log_string(s string) {
	if logging_disabled == FALSE {
		for i := 0; i < len(s); i++ {
			log_char(s[i])
		}
	}
}

func print_header() {
	if logging_disabled == FALSE {
		log_string("\nOther events:\n")
	}
	header_printed = TRUE
}

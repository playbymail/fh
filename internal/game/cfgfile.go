package game

// Port of cfgfile.c — config file parsing used by 'create species' and
// others. The C version accepts either a simple "key value" text format
// or (when the file name ends in ".json") a JSON array of objects parsed
// with the bundled cJSON library. The JSON support here uses
// encoding/json's token stream so that member order and duplicate keys
// behave like cJSON (cJSON_GetObjectItem returns the FIRST member whose
// name matches case-insensitively).

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"strings"
)

// species_cfg_t is from cfgfile.h. C string fields (char *) that are NULL
// when unset become empty Go strings.
type species_cfg_t struct {
	email     string
	govtname  string
	govttype  string
	homeworld string
	name      string
	ml        int
	gv        int
	ls        int
	bi        int

	experimental struct {
		econ_units   int
		make_bridges int
		ma_base      int
		mi_base      int
		ship_yards   int
		tech_bi      int
		tech_gv      int
		tech_ls      int
		tech_ma      int
		tech_mi      int
		tech_ml      int
	}
}

// key_value_t mirrors the C struct: kv.key == NULL becomes hasKey == false
// (end of input) and kv.value == NULL becomes hasValue == false.
type key_value_t struct {
	key      string
	value    string
	hasKey   bool
	hasValue bool
}

const cfgEOF = -1

// cfgFile wraps a reader with a C-style fgetc/feof so that cfgReadLine can
// reproduce the exact FILE* semantics of the original.
type cfgFile struct {
	f   *os.File
	r   *bufio.Reader
	eof bool // mirrors feof(fp)
}

func (fp *cfgFile) fgetc() int {
	c, err := fp.r.ReadByte()
	if err != nil {
		fp.eof = true
		return cfgEOF
	}
	return int(c)
}

// cfgReadLine reads one line from the input file and returns
// the key/value pair on the line.
// the key will have leading and trailing spaces removed.
// if there is no key on the line, the returned key will be an empty string.
// the value will have leading and trailing spaces removed.
// all interior runs of spaces will be condensed into a single space.
// if there is no value on the line, the returned value will be NULL.
// if there is a comment character ('#'), the remainder of the line is consumed.
// if the key and value are longer than the internal buffer,
// the remaining extra characters in the line are consumed but not returned.
// returns a key of NULL on end of input.
func cfgReadLine(fp *cfgFile) key_value_t {
	var buffer [128]byte
	eob := len(buffer) - 2 // should point to the last byte of buffer

	var kv key_value_t

	if fp == nil || fp.eof {
		return kv
	}
	kv.hasKey = true

	p := 0
	ch := fp.fgetc()
	for ; ch != '#' && ch != '\n' && ch != cfgEOF; ch = fp.fgetc() {
		// convert unwanted characters into spaces
		// (C: !isascii(ch) || ch == '\t' || ch == '\r' || isspace(ch) ||
		//  iscntrl(ch) || !isgraph(ch) — equivalent to "not graphic ASCII")
		if ch < 33 || ch > 126 {
			ch = ' '
		} else if ch == '"' || ch == '\\' || ch == '&' || ch == '<' || ch == '>' || ch == '!' || ch == ',' {
			ch = ' '
		}
		// ignore leading spaces and condense runs of spaces
		if ch == ' ' && (p == 0 || buffer[p-1] == ' ') {
			continue
		}
		// don't add past the end of the buffer
		if p != eob {
			buffer[p] = byte(ch)
			p++
		}
	}
	// consume the rest of the input line
	for ch != '\n' && ch != cfgEOF {
		ch = fp.fgetc()
	}
	// trim trailing spaces
	for p != 0 && buffer[p-1] == ' ' {
		p--
	}
	// split buffer into key and value based on the first space found in the buffer.
	// leave value set to NULL if there are no spaces.
	line := string(buffer[:p])
	if i := strings.IndexByte(line, ' '); i >= 0 {
		kv.key = line[:i]
		kv.value = line[i+1:]
		kv.hasValue = true
	} else {
		kv.key = line
	}
	return kv
}

func cfgSpeciesFree(c *species_cfg_t) *species_cfg_t {
	// memory management is handled by the garbage collector
	return nil
}

// cfgAtoi reproduces C atoi: optional leading whitespace, optional sign,
// then digits; parsing stops at the first non-digit and returns 0 if no
// digits were found.
func cfgAtoi(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\v' || s[i] == '\f' || s[i] == '\r') {
		i++
	}
	sign := 1
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}
	n := 0
	for i < len(s) && '0' <= s[i] && s[i] <= '9' {
		n = n*10 + int(s[i]-'0')
		i++
	}
	return sign * n
}

// cfgPerror mimics perror(prefix): "prefix: strerror(errno)". The C calls
// pass a prefix that already ends in ':', so the output contains '::'.
func cfgPerror(prefix string, err error) {
	msg := err.Error()
	var pe *fs.PathError
	if errors.As(err, &pe) {
		msg = pe.Err.Error()
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", prefix, msg)
}

func cfgSpeciesFromFile(name string) []*species_cfg_t {
	// reroute if the file name ends in ".json"
	isJSON := FALSE
	for i := 0; i < len(name); i++ {
		if name[i:] == ".json" {
			isJSON = TRUE
			break
		}
	}
	if isJSON != FALSE {
		return cfgSpeciesFromJSON(name)
	}

	slots := make([]*species_cfg_t, MAX_SPECIES+1)
	next := 0

	f, err := os.Open(name)
	if err != nil {
		cfgPerror("cfgSpeciesFromFile:", err)
		os.Exit(2)
	}
	fp := &cfgFile{f: f, r: bufio.NewReader(f)}
	line := 0
	var curr *species_cfg_t
	for {
		kv := cfgReadLine(fp)
		if !kv.hasKey {
			break
		}
		line++
		if len(kv.key) == 0 {
			continue
		} else if kv.key == "species" {
			// append a new section to the list
			curr = &species_cfg_t{}
			slots[next] = curr
			next++
			// fprintf(stderr, "cfgSpeciesFromFile: key '%s' val '%s'\n", kv.key, kv.value ? kv.value : "null");
		} else if curr == nil {
			fmt.Fprintf(os.Stderr, "error: %d: key %s: found key outside of section\n", line, kv.key)
			os.Exit(2)
		} else if kv.key == "bi" {
			val := 0
			if kv.hasValue {
				val = cfgAtoi(kv.value)
				if val < 1 || 15 < val {
					fmt.Fprintf(os.Stderr, "error: %d: key %s: value must be between 1 and 15\n", line, kv.key)
					os.Exit(2)
				}
			}
			curr.bi = val
		} else if kv.key == "email" {
			curr.email = ""
			if kv.hasValue {
				curr.email = kv.value
			}
		} else if kv.key == "govtname" {
			curr.govtname = ""
			if kv.hasValue {
				curr.govtname = kv.value
			}
		} else if kv.key == "govttype" {
			curr.govttype = ""
			if kv.hasValue {
				curr.govttype = kv.value
			}
		} else if kv.key == "gv" {
			val := 0
			if kv.hasValue {
				val = cfgAtoi(kv.value)
				if val < 1 || 15 < val {
					fmt.Fprintf(os.Stderr, "error: %d: key %s: value must be between 1 and 15\n", line, kv.key)
					os.Exit(2)
				}
			}
			curr.gv = val
		} else if kv.key == "homeworld" {
			curr.homeworld = ""
			if kv.hasValue {
				curr.homeworld = kv.value
			}
		} else if kv.key == "ls" {
			val := 0
			if kv.hasValue {
				val = cfgAtoi(kv.value)
				if val < 1 || 15 < val {
					fmt.Fprintf(os.Stderr, "error: %d: key %s: value must be between 1 and 15\n", line, kv.key)
					os.Exit(2)
				}
			}
			curr.ls = val
		} else if kv.key == "ml" {
			val := 0
			if kv.hasValue {
				val = cfgAtoi(kv.value)
				if val < 1 || 15 < val {
					fmt.Fprintf(os.Stderr, "error: %d: key %s: value must be between 1 and 15\n", line, kv.key)
					os.Exit(2)
				}
			}
			curr.ml = val
		} else if kv.key == "name" {
			// fprintf(stderr, "cfgSpeciesFromFile: key '%s' val '%s'\n", kv.key, kv.value ? kv.value : "null");
			curr.name = ""
			if kv.hasValue {
				curr.name = kv.value
			}
		} else {
			fmt.Fprintf(os.Stderr, "error: %d: key %s: unknown key\n", line, kv.key)
			os.Exit(2)
		}
	}
	// printf(" info: read %d lines from '%s'\n", line, name);

	return slots
}

// ---------------------------------------------------------------------------
// Minimal cJSON stand-in (order- and duplicate-preserving JSON tree).
// These helpers are private to this module; PORTING.md says the cjson/
// internals are not ported, so other modules must not rely on them.

const (
	cfgJSONFalse = iota
	cfgJSONTrue
	cfgJSONNull
	cfgJSONNumber
	cfgJSONString
	cfgJSONArray
	cfgJSONObject
)

type cfgJSON struct {
	kind  int
	key   string // member name when this node is a child of an object
	str   string
	num   float64
	child []*cfgJSON
}

func (j *cfgJSON) isArray() bool  { return j != nil && j.kind == cfgJSONArray }
func (j *cfgJSON) isObject() bool { return j != nil && j.kind == cfgJSONObject }
func (j *cfgJSON) isString() bool { return j != nil && j.kind == cfgJSONString }
func (j *cfgJSON) isNumber() bool { return j != nil && j.kind == cfgJSONNumber }
func (j *cfgJSON) isBool() bool {
	return j != nil && (j.kind == cfgJSONTrue || j.kind == cfgJSONFalse)
}

// valueint reproduces cJSON's valueint: doubles are truncated toward zero
// and clamped to [INT_MIN, INT_MAX]; booleans are 1 (true) or 0 (false).
func (j *cfgJSON) valueint() int {
	if j.kind == cfgJSONTrue {
		return 1
	}
	if j.kind != cfgJSONNumber {
		return 0
	}
	if j.num >= float64(math.MaxInt32) {
		return math.MaxInt32
	}
	if j.num <= float64(math.MinInt32) {
		return math.MinInt32
	}
	return int(j.num)
}

func cfgToLower(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

// getObjectItem reproduces cJSON_GetObjectItem: the first member whose
// name matches case-insensitively (byte-wise tolower), or nil.
func (j *cfgJSON) getObjectItem(name string) *cfgJSON {
	if !j.isObject() {
		return nil
	}
	for _, c := range j.child {
		if len(c.key) == len(name) {
			match := true
			for i := 0; i < len(name); i++ {
				if cfgToLower(c.key[i]) != cfgToLower(name[i]) {
					match = false
					break
				}
			}
			if match {
				return c
			}
		}
	}
	return nil
}

// cfgParseJSONValue builds a cfgJSON node from the decoder's token stream.
func cfgParseJSONValue(dec *json.Decoder) (*cfgJSON, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return cfgParseJSONToken(dec, tok)
}

func cfgParseJSONToken(dec *json.Decoder, tok json.Token) (*cfgJSON, error) {
	switch t := tok.(type) {
	case json.Delim:
		if t == '[' {
			node := &cfgJSON{kind: cfgJSONArray}
			for dec.More() {
				child, err := cfgParseJSONValue(dec)
				if err != nil {
					return nil, err
				}
				node.child = append(node.child, child)
			}
			if _, err := dec.Token(); err != nil { // consume ']'
				return nil, err
			}
			return node, nil
		}
		if t == '{' {
			node := &cfgJSON{kind: cfgJSONObject}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, _ := keyTok.(string)
				child, err := cfgParseJSONValue(dec)
				if err != nil {
					return nil, err
				}
				child.key = key
				node.child = append(node.child, child)
			}
			if _, err := dec.Token(); err != nil { // consume '}'
				return nil, err
			}
			return node, nil
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	case string:
		return &cfgJSON{kind: cfgJSONString, str: t}, nil
	case json.Number:
		f, _ := t.Float64()
		return &cfgJSON{kind: cfgJSONNumber, num: f}, nil
	case bool:
		if t {
			return &cfgJSON{kind: cfgJSONTrue}, nil
		}
		return &cfgJSON{kind: cfgJSONFalse}, nil
	default: // nil (JSON null)
		return &cfgJSON{kind: cfgJSONNull}, nil
	}
}

// cfgParseJSONFile mimics jsonParseFile from cjson/helpers.c: it returns
// nil if the file cannot be opened; on a parse error it reports the
// location ("file:line:col: error parsing just before\n\ttext") and exits.
// Trailing data after the top-level value is ignored, as with cJSON_Parse.
func cfgParseJSONFile(name string) *cfgJSON {
	f, err := os.Open(name)
	if err != nil {
		return nil
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		cfgPerror("json: parseFile: reading entire input", err)
		os.Exit(2)
	}

	// skip a UTF-8 byte order mark, as cJSON does
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	root, perr := cfgParseJSONValue(dec)
	if perr != nil {
		offset := dec.InputOffset()
		var se *json.SyntaxError
		if errors.As(perr, &se) {
			offset = se.Offset
		}
		line, col, text := cfgJSONErrorContext(data, offset)
		fmt.Fprintf(os.Stderr, "%s:%d:%d: error parsing just before\n\t%s\n", name, line, col, text)
		os.Exit(2)
	}
	return root
}

// cfgJSONErrorContext converts a byte offset into the line/column and
// nearby text reported by cJSON_GetError.
func cfgJSONErrorContext(data []byte, offset int64) (int, int, string) {
	line, col := 1, 1
	pos := 0
	for pos < len(data) && int64(pos) < offset && data[pos] != 0 {
		if data[pos] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		pos++
	}
	end := pos + 255
	if end > len(data) {
		end = len(data)
	}
	text := data[pos:end]
	if i := bytes.IndexByte(text, 0); i >= 0 {
		text = text[:i]
	}
	return line, col, string(text)
}

func cfgSpeciesFromJSON(filename string) []*species_cfg_t {
	array := cfgParseJSONFile(filename)
	if array == nil {
		fmt.Fprintf(os.Stderr, "error: %s: unable to read\n", filename)
		os.Exit(2)
	} else if !array.isArray() {
		fmt.Fprintf(os.Stderr, "error: %s: does not contain array\n", filename)
		os.Exit(2)
	}
	length := len(array.child)
	if length == 0 {
		fmt.Fprintf(os.Stderr, "error: %s: contains no data\n", filename)
		os.Exit(2)
	} else if length > MAX_SPECIES {
		fmt.Fprintf(os.Stderr, "error: %s: too many species\n", filename)
		fmt.Fprintf(os.Stderr, "       expect 0..%d species, got %d\n", MAX_SPECIES, length)
		os.Exit(2)
	}
	slots := make([]*species_cfg_t, length+1)

	idx := 0
	for _, elem := range array.child {
		if !elem.isObject() {
			fmt.Fprintf(os.Stderr, "error: %s: array must contain only objects\n", filename)
			os.Exit(2)
		}
		curr := &species_cfg_t{}
		slots[idx] = curr
		item := elem.getObjectItem("email")
		if item != nil && item.isString() {
			curr.email = item.str
		}
		item = elem.getObjectItem("name")
		if item != nil && item.isString() {
			curr.name = item.str
		}
		item = elem.getObjectItem("homeworld")
		if item != nil && item.isString() {
			curr.homeworld = item.str
		}
		item = elem.getObjectItem("govt-name")
		if item != nil && item.isString() {
			curr.govtname = item.str
		}
		item = elem.getObjectItem("govt-type")
		if item != nil && item.isString() {
			curr.govttype = item.str
		}
		item = elem.getObjectItem("tech-ml")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99) {
			curr.ml = item.valueint()
		}
		item = elem.getObjectItem("tech-gv")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99) {
			curr.gv = item.valueint()
		}
		// NOTE: the C original assigns "tech-ls" to curr->ml (not curr->ls);
		// the bug is preserved here for byte-identical behavior.
		item = elem.getObjectItem("tech-ls")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99) {
			curr.ml = item.valueint()
		}
		item = elem.getObjectItem("tech-bi")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99) {
			curr.bi = item.valueint()
		}
		item = elem.getObjectItem("x-bridges")
		if item != nil && item.isBool() {
			if item.valueint() != 0 {
				curr.experimental.make_bridges = TRUE
			} else {
				curr.experimental.make_bridges = FALSE
			}
		}
		item = elem.getObjectItem("x-bridges")
		if item != nil && item.isBool() {
			if item.valueint() != 0 {
				curr.experimental.make_bridges = TRUE
			} else {
				curr.experimental.make_bridges = FALSE
			}
		}
		item = elem.getObjectItem("x-econ-units")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99999999) {
			curr.experimental.econ_units = item.valueint()
		}
		item = elem.getObjectItem("x-ma-base")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99999999) {
			curr.experimental.ma_base = item.valueint()
		}
		item = elem.getObjectItem("x-mi-base")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99999999) {
			curr.experimental.mi_base = item.valueint()
		}
		// NOTE: the C original looks up "x-mi-base" a second time and assigns
		// it to curr->experimental.ma_base, clobbering any "x-ma-base" value;
		// the bug is preserved here for byte-identical behavior.
		item = elem.getObjectItem("x-mi-base")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99999999) {
			curr.experimental.ma_base = item.valueint()
		}
		item = elem.getObjectItem("x-ship-yards")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 99) {
			curr.experimental.ship_yards = item.valueint()
		}
		item = elem.getObjectItem("x-tech-bi")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 999) {
			curr.experimental.tech_bi = item.valueint()
		}
		item = elem.getObjectItem("x-tech-gv")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 999) {
			curr.experimental.tech_gv = item.valueint()
		}
		item = elem.getObjectItem("x-tech-ls")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 999) {
			curr.experimental.tech_ls = item.valueint()
		}
		item = elem.getObjectItem("x-tech-ma")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 999) {
			curr.experimental.tech_ma = item.valueint()
		}
		item = elem.getObjectItem("x-tech-mi")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 999) {
			curr.experimental.tech_mi = item.valueint()
		}
		item = elem.getObjectItem("x-tech-ml")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 999) {
			curr.experimental.tech_ml = item.valueint()
		}
		item = elem.getObjectItem("x-tech-ls")
		if item != nil && item.isNumber() && (0 <= item.valueint() && item.valueint() <= 999) {
			curr.experimental.tech_ls = item.valueint()
		}
		idx++
	}
	return slots
}

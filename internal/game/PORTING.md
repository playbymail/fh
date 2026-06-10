# Porting conventions for internal/game

This package is a faithful port of the Far Horizons C engine
(`../Far-Horizons/src/*.c`). Every porter (human or agent) MUST follow
these rules so the modules integrate cleanly and the Go engine produces
byte-identical results to the C engine for a fixed `FH_SEED`.

## Layout

One Go file per C file: `jump.c` → `jump.go`, `do.c` → `do_*.go` (split
allowed). Keep the original comment text and code order within the file.

## Naming

- Keep C function and variable names exactly, including snake_case:
  `do_JUMP_command`, `get_class_abbr`, `save_species_data`.
- Core structs use the C typedef names: `galaxy_data_t`, `star_data_t`,
  `planet_data_t`, `nampla_data_t`, `ship_data_t`, `species_data_t`,
  `sp_loc_data_t`, `trans_data_t` (see `types.go`).
- The C field `type` is renamed: `star->type` → `.star_type`,
  `ship->type` → `.ship_type`, `transaction.type` → `.trans_type`.
- Everything stays unexported. CLI entry points get exported wrappers
  later in one place.

## Globals

All cross-module globals are already declared in `vars.go` (plus
`money.go`, `log.go`, `prng.go`). NEVER redeclare them and NEVER edit the
shared files (`vars.go`, `types.go`, `const.go`, `tables.go`, `engine.go`,
`log.go`, `money.go`, `ship.go`, `species.go`, `nampla.go`, `item.go`,
`dev_log.go`, `prng.go`). Variables that are file-local in C (statics, or
declared only in the C file you are porting, like combat.c's globals) are
declared in YOUR Go file. If they need re-initialization between runs,
add a `reset<Module>State()` function in your file and note it in your
report.

Current-entity globals (`species`, `nampla`, `ship`, `star`, `planet`,
`home_planet`, plus index variables) and the location globals `x, y, z,
pn` exist exactly as in C. C code that declares locals shadowing them
(e.g. `int x, y, z;` inside a function) should declare locals in Go too,
matching the C scoping.

## Pointer arithmetic

C iterates contiguous arrays with pointers:

    nampla = nampla_base - 1;
    for (i = 0; i < species->num_namplas; i++) {
        nampla++;
        ...

`star_base`, `planet_base`, `nampla_base`, `ship_base`, `namp_data[i]`,
`ship_data[i]` are Go slices of pointers (`[]*star_data_t` etc.).
Translate to indexing:

    for i := 0; i < species.num_namplas; i++ {
        nampla = nampla_base[i]
        ...

When C assigns the global mid-loop (as above), assign the global in Go
too — later code often relies on it. Pointer comparisons (`nampla ==
other`) work the same on Go pointers. `nampla - nampla_base` (index of a
record) must be tracked with an index variable or by searching the slice
(prefer carrying the index alongside).

C reallocation idioms (`ncalloc` + copy, `extra_namplas` headroom) become
`append`/new slices; growing a slice keeps existing pointers valid
because elements are themselves pointers.

## Booleans and ints

The C engine uses `int` with TRUE/FALSE (constants 1/0 here). Keep int
flags and compare with `!= FALSE` / `== FALSE` to preserve semantics
(some C code sets flags to values other than 1). C `short`/`long`/`char`
counters all become Go `int`.

## Strings

Fixed C char buffers are Go strings. `strcpy(a, b)` → `a = b`;
`strcat` → `+=`; `sprintf(buf, ...)` → `buf = fmt.Sprintf(...)`.
C string comparisons with `strcmp(a,b) == 0` → `a == b`. Functions
returning `char *` static buffers return Go strings (and also set the
global if C did, e.g. `ship_name` sets `full_ship_id`).
For parsing code that walks `char *` pointers, use an index into the
string and the convention that reading at/past `len(s)` yields 0 (NUL);
see `agrep_score` in `engine.go` for the pattern.

## Files and I/O

- Output streams (`log_file`, `orders_file`, `summary_file`, report
  files) are `*os.File`; `fprintf` → `fmt.Fprintf`.
- Input files use the `cfile` wrapper in `engine.go`: `fopen_r(name)`
  (nil on failure, like fopen), `fp.fgets(n)`, `readln(fp, n)`,
  `fp.fclose()`.
- Binary .dat files must keep the exact on-disk layout of the
  `binary_*_data_t` structs in `../Far-Horizons/src/data.h`,
  little-endian, including reserved/padding fields (zeros). Counts are
  `int32`. Fixed-size name fields are NUL-padded byte arrays.
- On fatal errors the C engine prints to stderr and calls `exit(-1)`;
  port as `fmt.Fprintf(os.Stderr, ...)` + `os.Exit(255)` (Go cannot exit
  with -1). `exit(0)` → `os.Exit(0)`, `exit(2)` → `os.Exit(2)`.

## Randomness

ALL randomness goes through `rnd(max)` (returns 1..max inclusive,
`prng.go`). Never use math/rand. Preserve the exact number AND order of
`rnd` calls — determinism against the C engine depends on it.

## Logging

`log_string`, `log_char`, `log_int`, `log_long`, `log_printf`,
`log_message`, `print_header` are ported in `log.go` and reproduce the
C line-wrapping exactly. Use them; don't write to stdout directly except
where the C code uses printf (gamemaster console output).

## Cross-module calls

If the C code calls a function owned by another module that is not yet
ported (e.g. `get_command()`, `save_species_data()`), call it by its
exact C name and DO NOT define a stub. The package may not compile until
all modules of the wave land; that is expected. Make sure your own file
is gofmt-clean and internally consistent.

## CLI entry points

Each C module's `*Command(int argc, char *argv[])` entry function (e.g.
`jumpCommand`, `reportCommand`) is ported as

    func jumpCommand(args []string) int

where `args[0]` is the command name, mirroring argv, and the return value
is the process exit code. Keep the C argument parsing (including the
`--opt=value` splitting loop) and usage/error messages byte-identical.
The ff-based CLI in main.go passes raw arguments through to these.

## What not to port

- `ncalloc`, `memsafe.c`, `mask.c` (memory helpers) — use Go allocation.
- `add_tran.c` — dead legacy code (not in the CMake build).
- `cjson/`, `sexpr.c` marshaling internals — JSON uses encoding/json
  against the same field names as the C JSON exporter where needed.

# ultron-lifecycle.sh — the single source of process truth for the Ultron
# turn-folder lifecycle. Sourced (not executed) by the lifecycle scripts so the
# state predicate and transition guards live in exactly ONE place; no script
# re-derives the rule. See docs/project-ultron/turn-lifecycle.md.
#
# Forward-looking: these same operations are destined to live behind a Go
# interface the scripting engine defines and `fhc` (and later `fh`) implement, so
# the scripting engine codes against the interface, not a concrete engine. This
# shell library is the validation-phase stand-in for that interface — prove the
# process here, then move the rules behind the interface unchanged.
#
# Requires bash. The engine binary defaults to dist/local/fhc but can be
# overridden with ULTRON_FHC=/path/to/fhc for testing.

# Resolve the repo root from this library's own location (tools/lib -> repo root).
ULTRON_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ULTRON_REPO_ROOT=$(cd "$ULTRON_LIB_DIR/../.." && pwd)
ULTRON_FHC="${ULTRON_FHC:-$ULTRON_REPO_ROOT/dist/local/fhc}"

# ultron_log / ultron_die — uniform progress + error reporting to stderr.
ultron_log() { printf 'ultron: %s\n' "$*" >&2; }
ultron_die() { printf 'ultron: error: %s\n' "$*" >&2; exit 1; }

# ultron_resolve_fhc — ensure the prebuilt engine binary exists (the engine
# writes to its cwd, so the lifecycle scripts cd into a turn dir and cannot use
# `go run ./cmd/fhc` from the repo root).
ultron_resolve_fhc() {
	if [ ! -x "$ULTRON_FHC" ]; then
		ultron_die "engine binary not found at $ULTRON_FHC — build it: (cd $ULTRON_REPO_ROOT && go build -o dist/local/fhc ./cmd/fhc)"
	fi
}

# ultron_turns <data-root> — echo the integer-named turn folders, ascending, one
# per line. Non-integer entries and plain files are ignored (mirrors the
# fh.load{} scan). Empty output if the data-root has no turn folders.
ultron_turns() {
	local root=$1 path name
	[ -d "$root" ] || return 0
	for path in "$root"/*/; do
		[ -d "$path" ] || continue # guards the no-match literal glob
		name=${path%/}
		name=${name##*/}
		case "$name" in
			'' | *[!0-9]*) continue ;; # non-integer name -> not a turn
		esac
		printf '%s\n' "$name"
	done | sort -n
}

# ultron_active_turn <data-root> — echo the highest-numbered (active) turn
# folder, or nothing if there are no turn folders. Everything below the active
# turn is frozen history.
ultron_active_turn() {
	ultron_turns "$1" | tail -n 1
}

# ultron_turn_number <data-root> <N> — echo galaxy.dat's authoritative
# turn_number for folder N (the count of turns resolved so far). Dies if folder N
# has no galaxy.dat.
ultron_turn_number() {
	local root=$1 n=$2 out
	[ -f "$root/$n/galaxy.dat" ] || ultron_die "no galaxy.dat in $root/$n"
	out=$(cd "$root/$n" && "$ULTRON_FHC" show turn_number) || ultron_die "could not read turn_number in $root/$n"
	out=$(printf '%s' "$out" | tr -dc '0-9')
	case "$out" in
		'' | *[!0-9]*) ultron_die "unexpected turn_number output '$out' in $root/$n" ;;
	esac
	printf '%s' "$out"
}

# ultron_turn_state <data-root> <N> — THE lifecycle predicate. Echoes one of:
#   absent    — folder N has no galaxy.dat (not a game / not yet created)
#   pending   — turn_number == N-1 (orders open, pipeline not yet run)
#   resolved  — turn_number == N   (pipeline has run; reports written)
#   anomalous — neither (corrupt or hand-edited; callers should refuse to act)
# This is the one rule every lifecycle script consults; see the doc's state
# machine. (Turn 0 is born resolved, so it is only ever absent/resolved.)
ultron_turn_state() {
	local root=$1 n=$2 tn
	if [ ! -f "$root/$n/galaxy.dat" ]; then
		printf 'absent'
		return 0
	fi
	tn=$(ultron_turn_number "$root" "$n")
	if [ "$tn" -eq "$n" ]; then
		printf 'resolved'
	elif [ "$tn" -eq $((n - 1)) ]; then
		printf 'pending'
	else
		printf 'anomalous'
	fi
}

# ultron_require_state <data-root> <N> <want> — die unless folder N is in the
# wanted state. The transition guards the GM control verbs are built from:
#   genesis            -> require turn 0 'absent'   (refuse to overwrite)
#   run-this-turn(N)   -> require N 'pending'       (refuse an already-run turn)
#   freeze-and-forward -> require N 'resolved'      (don't forward a pending turn)
ultron_require_state() {
	local root=$1 n=$2 want=$3 got
	got=$(ultron_turn_state "$root" "$n")
	[ "$got" = "$want" ] || ultron_die "turn $n must be '$want' but is '$got'"
}

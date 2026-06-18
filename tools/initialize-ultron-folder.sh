#!/usr/bin/env bash
#
# initialize-ultron-folder.sh — create a fresh Far Horizons game inside an
# existing Ultron data-root, as turn 0 (the genesis / pre-turn-1 setup state).
#
# Usage:
#   tools/initialize-ultron-folder.sh <data-root> [seed] [num-species]
#
#   <data-root>    Path to the Ultron folder, e.g. data/alpha. MUST already
#                  exist — this script never creates the Ultron folder itself,
#                  only the turn-0 game directory inside it.
#   [seed]         FH_SEED for deterministic generation. Default: 12345.
#   [num-species]  Number of player species to create (1..100). Default: 5.
#
# Layout produced (game-id is dropped; the data-root IS the single game):
#
#   <data-root>/
#     0/                 # turn 0 — the genesis state this script writes
#       galaxy.dat stars.dat planets.dat sp01.dat … spNN.dat
#       species.cfg      # the generated roster, kept for reference
#       homesystem*.dat  # home-system templates
#
# Turns 1, 2, 3, … are produced later by running the turn pipeline; those are
# the integer turn dirs fh.load{} scans. Turn 0 is the genesis snapshot.
#
# The engine reads/writes bare filenames in its current directory, so each fhc
# command runs with cwd set to the turn-0 dir. That makes `go run ./cmd/fhc`
# (which must run from the repo root) unusable here, so this script uses the
# prebuilt binary at dist/local/fhc. Build it first:
#
#   go build -o dist/local/fhc ./cmd/fhc
#
# Run from the repo root so a relative <data-root> like data/alpha resolves.
# If the game already exists (turn-0 galaxy.dat present), the script logs and
# exits without overwriting it.

set -euo pipefail

DEFAULT_SEED=12345
DEFAULT_NUM_SPECIES=5
MAX_SPECIES=100 # mirrors internal/game/const.go MAX_SPECIES

# Resolve the repo root from this script's location (tools/ is at the root).
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
FHC="$REPO_ROOT/dist/local/fhc"

log()  { printf 'init-ultron: %s\n' "$*" >&2; }
die()  { printf 'init-ultron: error: %s\n' "$*" >&2; exit 1; }

usage() {
	printf 'usage: %s <data-root> [seed] [num-species]\n' "$(basename "$0")" >&2
	printf '  <data-root>    existing Ultron folder, e.g. data/alpha (required)\n' >&2
	printf '  [seed]         FH_SEED for generation (default %s)\n' "$DEFAULT_SEED" >&2
	printf '  [num-species]  number of species, 1..%d (default %d)\n' "$MAX_SPECIES" "$DEFAULT_NUM_SPECIES" >&2
}

# --- parse + validate arguments ------------------------------------------------

if [ "$#" -lt 1 ] || [ "$#" -gt 3 ]; then
	usage
	exit 1
fi

DATA_ROOT_ARG=$1
SEED=${2:-$DEFAULT_SEED}
NUM_SPECIES=${3:-$DEFAULT_NUM_SPECIES}

case "$SEED" in
	'' | *[!0-9]*) die "seed must be a non-negative integer (got '$SEED')" ;;
esac
case "$NUM_SPECIES" in
	'' | *[!0-9]*) die "num-species must be a positive integer (got '$NUM_SPECIES')" ;;
esac
if [ "$NUM_SPECIES" -lt 1 ] || [ "$NUM_SPECIES" -gt "$MAX_SPECIES" ]; then
	die "num-species must be between 1 and $MAX_SPECIES (got $NUM_SPECIES)"
fi

if [ ! -x "$FHC" ]; then
	die "engine binary not found at $FHC — build it first: (cd $REPO_ROOT && go build -o dist/local/fhc ./cmd/fhc)"
fi

# The Ultron folder must already exist; we never create it (only the turn dir).
if [ ! -d "$DATA_ROOT_ARG" ]; then
	die "data-root '$DATA_ROOT_ARG' does not exist — create the Ultron folder first; this script will not"
fi
DATA_ROOT=$(cd "$DATA_ROOT_ARG" && pwd)
TURN0="$DATA_ROOT/0"

# Refuse to overwrite an existing game.
if [ -f "$TURN0/galaxy.dat" ]; then
	log "game already exists at $TURN0/galaxy.dat — refusing to overwrite; nothing to do"
	exit 0
fi

# --- create the game -----------------------------------------------------------

mkdir -p "$TURN0"
export FH_SEED="$SEED"

# fhc runs the engine with cwd set to the turn-0 dir so it reads/writes its bare
# filenames there. Output is left visible so a failure is easy to diagnose.
fhc() { ( cd "$TURN0" && "$FHC" "$@" ); }

log "creating game in $TURN0 (seed=$SEED, species=$NUM_SPECIES)"

log "1/3 create galaxy"
fhc create galaxy --species="$NUM_SPECIES" >/dev/null

# The galaxy radius is derived from the species count; the species' minimum
# home-system separation must be <= radius/2. Read the actual radius back and
# pick a conservative fraction (radius/3) so home placement stays reliable even
# for small galaxies — a smaller minimum separation is always easier to satisfy.
RADIUS=$(fhc show radius | tr -dc '0-9')
case "$RADIUS" in
	'' | *[!0-9]*) die "could not read galaxy radius (got '$RADIUS')" ;;
esac
SP_RADIUS=$(( RADIUS / 3 ))
[ "$SP_RADIUS" -lt 1 ] && SP_RADIUS=1

log "2/3 create home-system-templates"
fhc create home-system-templates >/dev/null

# Generate a valid species.cfg: unique 5+ char names, govttype/govtname/homeworld
# within the engine's 5..31 length limits, and tech levels (ml+gv+ls+bi) summing
# to < 16 (MA/MI default to 10 each, added by the engine).
CFG="$TURN0/species.cfg"
: > "$CFG"
for i in $(seq 1 "$NUM_SPECIES"); do
	printf 'species\n' >> "$CFG"
	printf '    name      Species %02d\n'           "$i" >> "$CFG"
	printf '    homeworld Homeworld %02d\n'         "$i" >> "$CFG"
	printf '    govtname  Government %02d\n'        "$i" >> "$CFG"
	printf '    govttype  Republic\n'                   >> "$CFG"
	printf '    ml 3\n    gv 1\n    ls 1\n    bi 3\n'    >> "$CFG"
	printf '    email     species%02d@example.com\n\n' "$i" >> "$CFG"
done

log "3/3 create species (galaxy radius=$RADIUS, min home separation=$SP_RADIUS)"
fhc create species --config=species.cfg --radius="$SP_RADIUS" >/dev/null

log "done: turn-0 game ready at $TURN0 ($NUM_SPECIES species)"

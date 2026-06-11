#!/bin/sh
# Regenerates the C-engine reference data used to validate the Go port.
#
# Requires the C engine built at ../Far-Horizons/build/fh (cmake -S . -B
# build && cmake --build build). Everything is seeded with FH_SEED so the
# run is reproducible; the Go engine must produce byte-identical .dat
# files, logs, and reports at every step.
#
# Snapshots are written next to this script, one directory per stage:
#   galaxy/  just `create galaxy`
#   setup/   galaxy + templates + species, before any turn
#   turn1/   after locations, default orders, and a full turn 1 pipeline
#   turn2/   after a second turn (orders regenerated) from the turn1 state
#   turn3/   after a third turn, continuing the same run directory
#   turn4/   after a fourth turn, continuing the same run directory
#
# Override the engine path with FH=/path/to/fh. The reference data is
# git-ignored (see testdata/.gitignore); run this to (re)create it locally.
set -e

HERE=$(cd "$(dirname "$0")" && pwd)
FH=${FH:-$HERE/../../../Far-Horizons/build/fh}
SEED=1924085713
EXAMPLES=$(dirname "$FH")/../examples

if [ ! -x "$FH" ]; then
	echo "error: C engine not found at $FH" >&2
	echo "build it: (cd ../Far-Horizons && cmake -S . -B build && cmake --build build)" >&2
	exit 1
fi

export FH_SEED=$SEED

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
cd "$WORK"

# snapshot copies the entire working directory into a fresh stage dir.
snapshot() {
	rm -rf "$HERE/$1"
	mkdir -p "$HERE/$1"
	cp ./* "$HERE/$1/"
}

# --- galaxy: just create the galaxy ---
"$FH" create galaxy --species=9 > create_galaxy.log 2>&1
snapshot galaxy

# --- setup: add home-system templates and the species ---
"$FH" create home-system-templates > create_templates.log 2>&1
cp "$EXAMPLES/species.cfg" .
"$FH" create species --config=species.cfg --radius=6 > create_species.log 2>&1
# logrnd is state-independent (it seeds from the historical default seed),
# so capture it here once and keep it with the setup snapshot.
"$FH" logrnd > logrnd.log 2>&1
# version is state-independent too; it prints the engine version and exits 2
# (a quirk of version.c), so guard the non-zero status against `set -e`.
"$FH" version > version.log 2>&1 || true
# inspect is state-independent; it prints a fixed table of C sizeof() values.
"$FH" inspect > inspect.log 2>&1
snapshot setup

# --- turn1: locations, default orders, and the full turn pipeline ---
cp "$EXAMPLES/noorders.txt" .
"$FH" locations > locations.log 2>&1
"$FH" create orders > create_orders.log 2>&1
for sp in 01 02 03 04; do cp sp$sp.ord sp$sp.t1.ord; done

"$FH" combat > combat.log 2>&1
"$FH" pre-departure > predeparture.log 2>&1
"$FH" jump > jump.log 2>&1
"$FH" production > production.log 2>&1
"$FH" post-arrival > postarrival.log 2>&1
"$FH" finish > finish.log 2>&1
"$FH" report > report.log 2>&1
"$FH" stats > stats.log 2>&1
"$FH" turn > turn.log 2>&1
# list/show are read-only reports over the post-turn-1 state; capture a few
# variants covering each output branch. list galaxy (default, with planets),
# the wormhole branch, and the per-species scanned branch.
"$FH" list galaxy > list_galaxy.log 2>&1
"$FH" list galaxy --planets=false > list_galaxy_nopl.log 2>&1
"$FH" list galaxy --wormholes=true > list_galaxy_worm.log 2>&1
"$FH" list scanned --species=1 > list_scanned_sp1.log 2>&1
"$FH" list scanned --species=2 > list_scanned_sp2.log 2>&1
# show: game-value queries, the ASCII map, the paged map (also writes
# galaxy.map, preserved as show_galaxy.map for the file-comparison test), and
# the help text.
"$FH" show num_stars num_species num_planets num_natural_wormholes radius d_num_species turn_number > show_values.log 2>&1
"$FH" show galaxy --ascii > show_galaxy_ascii.log 2>&1
"$FH" show galaxy > show_galaxy.log 2>&1
cp galaxy.map show_galaxy.map
"$FH" show help > show_help.log 2>&1
snapshot turn1

# run_turn N runs one later turn (N >= 2) of the pipeline in the same
# accumulating work directory, then snapshots it into turnN/. The combat
# (etc.) phases do not consume sp0X.ord, so the previous turn's orders are
# removed first to force `create orders` to regenerate fresh defaults from
# the current state. Command order matches turn 1 except that `create
# orders` runs before `locations` (turn 1 generated locations from setup
# before any orders existed). Logs/locations/transactions accumulate.
run_turn() {
	n=$1
	rm -f sp01.ord sp02.ord sp03.ord sp04.ord
	"$FH" create orders > create_orders.t$n.log 2>&1
	for sp in 01 02 03 04; do cp sp$sp.ord sp$sp.t$n.ord; done

	"$FH" locations > locations.t$n.log 2>&1
	"$FH" combat > combat.t$n.log 2>&1
	"$FH" pre-departure > predeparture.t$n.log 2>&1
	"$FH" jump > jump.t$n.log 2>&1
	"$FH" production > production.t$n.log 2>&1
	"$FH" post-arrival > postarrival.t$n.log 2>&1
	"$FH" finish > finish.t$n.log 2>&1
	"$FH" report > report.t$n.log 2>&1
	"$FH" stats > stats.t$n.log 2>&1
	"$FH" turn > turn.t$n.log 2>&1
	snapshot turn$n
}

run_turn 2
run_turn 3
run_turn 4

echo "reference data written to $HERE/{galaxy,setup,turn1,turn2,turn3,turn4}"

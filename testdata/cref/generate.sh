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
snapshot turn1

# --- turn2: regenerate default orders and run the pipeline again ---
# combat (etc.) do not consume sp0X.ord, so remove the turn-1 orders to
# force `create orders` to regenerate fresh defaults from the turn-1 state.
rm -f sp01.ord sp02.ord sp03.ord sp04.ord
"$FH" create orders > create_orders.t2.log 2>&1
for sp in 01 02 03 04; do cp sp$sp.ord sp$sp.t2.ord; done

"$FH" locations > locations.t2.log 2>&1
"$FH" combat > combat.t2.log 2>&1
"$FH" pre-departure > predeparture.t2.log 2>&1
"$FH" jump > jump.t2.log 2>&1
"$FH" production > production.t2.log 2>&1
"$FH" post-arrival > postarrival.t2.log 2>&1
"$FH" finish > finish.t2.log 2>&1
"$FH" report > report.t2.log 2>&1
"$FH" stats > stats.t2.log 2>&1
"$FH" turn > turn.t2.log 2>&1
snapshot turn2

echo "reference data written to $HERE/{galaxy,setup,turn1,turn2}"

#!/usr/bin/env bash
#
# run-this-turn.sh — resolve the active turn. The third lifecycle verb; see
# docs/project-ultron/turn-lifecycle.md.
#
# Usage:
#   tools/run-this-turn.sh <data-root>
#
#   <data-root>    Path to the Ultron folder, e.g. data/alpha (must hold a game
#                  with an active PENDING turn — freeze-and-forward opens one).
#
# Takes the active turn N (the highest-numbered folder), requires it to be
# PENDING (galaxy.dat turn_number == N-1), runs `fhc` to completion in N/, and
# advances galaxy.dat from N-1 to N (RESOLVED). `report` — not `finish` — writes
# the spNN.rpt.t<turn> files, so the pipeline runs through report (plus stats /
# turn summaries).
#
# Order staging. The engine reads flat sp<NN>.ord files in its cwd. This script
# rebuilds those from the per-species staging slots first: it removes any stale
# orders carried forward by freeze-and-forward, copies each <species>/orders
# (species id is a bare integer, 1..MAX_SPECIES) to the flat sp<NN>.ord, and then
# `create orders` fills DEFAULT orders for every species that did not submit
# (using the bundled noorders.txt template). With no staging slots yet, that
# means a full default-order turn — exactly the start-of-game case.
#
# Run from the repo root so a relative <data-root> resolves. The engine binary
# and the lifecycle predicate come from tools/lib/ultron-lifecycle.sh.

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=tools/lib/ultron-lifecycle.sh
. "$SCRIPT_DIR/lib/ultron-lifecycle.sh"

NOORDERS_TEMPLATE="$SCRIPT_DIR/noorders.txt"

usage() {
	printf 'usage: %s <data-root>\n' "$(basename "$0")" >&2
	printf '  <data-root>    Ultron folder with an active pending turn, e.g. data/alpha\n' >&2
}

# --- parse + validate arguments ------------------------------------------------

if [ "$#" -ne 1 ]; then
	usage
	exit 1
fi

DATA_ROOT_ARG=$1

ultron_resolve_fhc

if [ ! -d "$DATA_ROOT_ARG" ]; then
	ultron_die "data-root '$DATA_ROOT_ARG' does not exist"
fi
DATA_ROOT=$(cd "$DATA_ROOT_ARG" && pwd)

if [ ! -f "$NOORDERS_TEMPLATE" ]; then
	ultron_die "missing orders template $NOORDERS_TEMPLATE"
fi

# The active turn is the highest-numbered folder; there must be a game to run.
N=$(ultron_active_turn "$DATA_ROOT")
if [ -z "$N" ]; then
	ultron_die "no game in '$DATA_ROOT' — run initialize-ultron-folder.sh first"
fi

# Pre-condition: the active turn must be pending. Re-running a resolved turn is
# the idempotence guard (mirrors genesis's refuse-to-overwrite).
STATE=$(ultron_turn_state "$DATA_ROOT" "$N")
case "$STATE" in
	pending) ;;
	resolved) ultron_die "turn $N is already resolved — run freeze-and-forward to open the next turn" ;;
	*) ultron_die "turn $N is '$STATE' — refusing to run" ;;
esac

TURN_DIR="$DATA_ROOT/$N"

# --- stage orders --------------------------------------------------------------

# Start from a clean flat order namespace: drop any sp*.ord carried forward from
# the prior turn so they cannot be mistaken for this turn's submissions.
( cd "$TURN_DIR" && rm -f sp*.ord )

# Materialize each per-species staging slot (<turn>/<species>/orders) into the
# flat sp<NN>.ord the engine expects. species id is a bare integer.
staged=0
for sp in $(ultron_species_dirs "$DATA_ROOT" "$N"); do
	src="$TURN_DIR/$sp/orders"
	if [ -f "$src" ]; then
		printf -v flat 'sp%02d.ord' "$sp"
		cp "$src" "$TURN_DIR/$flat"
		staged=$((staged + 1))
	fi
done

# create orders needs its template in cwd; it fills defaults for un-staged species.
cp "$NOORDERS_TEMPLATE" "$TURN_DIR/noorders.txt"

ultron_log "running turn $N in $TURN_DIR ($staged species staged orders; rest defaulted)"

# --- run the pipeline to completion -------------------------------------------

# fhc runs the engine with cwd set to the turn dir. Output is left visible so a
# failure is easy to diagnose.
fhc() { ( cd "$TURN_DIR" && "$ULTRON_FHC" "$@" ); }

# Canonical turn pipeline (see the design doc and testdata/cref/generate.sh).
# locations first, then default-fill orders, then the resolution phases; report
# writes the player reports; stats/turn are summaries.
fhc locations >/dev/null
fhc create orders >/dev/null
fhc combat >/dev/null
fhc pre-departure >/dev/null
fhc jump >/dev/null
fhc production >/dev/null
fhc post-arrival >/dev/null
fhc finish >/dev/null
fhc report >/dev/null
fhc stats >/dev/null
fhc turn >/dev/null

# Post-condition: the turn must now read RESOLVED (turn_number advanced to N).
NEW_STATE=$(ultron_turn_state "$DATA_ROOT" "$N")
if [ "$NEW_STATE" != "resolved" ]; then
	ultron_die "internal error: turn $N is '$NEW_STATE' after run, expected 'resolved'"
fi

ultron_log "done: turn $N resolved (turn_number=$N); reports written in $TURN_DIR"
ultron_log "open the next turn with: tools/freeze-and-forward.sh $DATA_ROOT_ARG"

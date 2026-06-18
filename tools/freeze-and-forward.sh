#!/usr/bin/env bash
#
# freeze-and-forward.sh — open the next turn. The second lifecycle verb; see
# docs/project-ultron/turn-lifecycle.md.
#
# Usage:
#   tools/freeze-and-forward.sh <data-root>
#
#   <data-root>    Path to the Ultron folder, e.g. data/alpha (must exist and
#                  already hold a game — run initialize-ultron-folder.sh first).
#
# Takes the current active turn N (the highest-numbered folder), requires it to
# be RESOLVED (its pipeline has run, galaxy.dat turn_number == N), and creates
# N+1/ as a byte copy of N/. The copy carries N's turn_number, so N+1 is born
# PENDING (turn_number == N == (N+1)-1): orders open, not yet run.
#
# "Freeze" is procedural, not a filesystem marker (see the doc's single-source-
# of-process-truth rule): once N+1 exists, N is no longer the active turn, so the
# mutating verbs refuse to touch it — it becomes query-only. This script writes
# no freeze sentinel; creating N+1 *is* the freeze.
#
# Orders for the new turn go into N+1/<species>/ and are placed by the future
# order-staging tool; the per-species subdir format is still undecided, so this
# script does not create them. run-this-turn handles fresh order generation when
# the turn is resolved.
#
# Run from the repo root so a relative <data-root> resolves. The engine binary
# and the lifecycle predicate come from tools/lib/ultron-lifecycle.sh.

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=tools/lib/ultron-lifecycle.sh
. "$SCRIPT_DIR/lib/ultron-lifecycle.sh"

usage() {
	printf 'usage: %s <data-root>\n' "$(basename "$0")" >&2
	printf '  <data-root>    existing Ultron folder holding a game, e.g. data/alpha\n' >&2
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

# The active turn is the highest-numbered folder; there must be a game to forward.
N=$(ultron_active_turn "$DATA_ROOT")
if [ -z "$N" ]; then
	ultron_die "no game in '$DATA_ROOT' — run initialize-ultron-folder.sh first"
fi

# Pre-condition: the active turn must be resolved before we forward it. Forwarding
# a pending turn would copy a half-finished, un-run state into the next folder.
STATE=$(ultron_turn_state "$DATA_ROOT" "$N")
case "$STATE" in
	resolved) ;;
	pending) ultron_die "turn $N is pending (not yet run) — run-this-turn before forwarding" ;;
	*) ultron_die "turn $N is '$STATE' — refusing to forward" ;;
esac

NEXT=$((N + 1))
TARGET="$DATA_ROOT/$NEXT"

# N is active, so N+1 should not exist; guard against clobbering anything that does.
if [ -e "$TARGET" ]; then
	ultron_die "turn $NEXT already exists at $TARGET — refusing to overwrite"
fi

# --- freeze N + forward to N+1 -------------------------------------------------

ultron_log "freezing turn $N; opening turn $NEXT"
mkdir -p "$TARGET"
# Byte copy of the whole turn folder: the engine never sees folder names, so the
# copied flat working dir is identical to one produced in place (parity-safe).
cp -R "$DATA_ROOT/$N/." "$TARGET/"

# Post-condition: the new turn must read PENDING (carries N's turn_number in a
# folder named N+1), and it must now be the active turn (N is frozen behind it).
NEW_STATE=$(ultron_turn_state "$DATA_ROOT" "$NEXT")
if [ "$NEW_STATE" != "pending" ]; then
	ultron_die "internal error: turn $NEXT is '$NEW_STATE' after forward, expected 'pending'"
fi
NEW_ACTIVE=$(ultron_active_turn "$DATA_ROOT")
if [ "$NEW_ACTIVE" != "$NEXT" ]; then
	ultron_die "internal error: active turn is '$NEW_ACTIVE' after forward, expected '$NEXT'"
fi

ultron_log "done: turn $N frozen (query-only); turn $NEXT is pending and active"
ultron_log "stage orders for turn $NEXT, then run: tools/run-this-turn.sh $DATA_ROOT_ARG"

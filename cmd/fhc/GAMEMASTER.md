# Running a Far Horizons game

A practical guide for the **game-master (GM)** who wants to start a new game of
Far Horizons, set it up, collect orders by e-mail, run each turn, mail out the
reports, and roll forward to the next turn.

This guide assumes you are on **macOS or Linux**, you have **git** installed,
and you are comfortable at the command line. It uses the byte-faithful engine
(`fhc`) — the trusted reference implementation — and a **git branch per turn**
so every turn is committed, reviewable, and reversible.

> Throughout, the engine binary is called `fhc`. The program prints its own
> name as `fh` in usage/help text; that is cosmetic — type `fhc`.

---

## 1. Prerequisites

- **Go** (to build the engine) — <https://go.dev/dl/>
- **git** — `git --version` should print a version
- A way to receive orders by e-mail and copy the attachments/text onto disk
- A text editor

Build the engine once from the repository root:

```sh
go build -o dist/local/fhc ./cmd/fhc
```

Then put it somewhere on your `PATH` so you can run it from your game
directory. For example:

```sh
mkdir -p ~/bin
cp dist/local/fhc ~/bin/fhc
# make sure ~/bin is on your PATH (add to ~/.zshrc or ~/.bashrc if needed):
#   export PATH="$HOME/bin:$PATH"
```

Confirm it works:

```sh
fhc version      # prints the engine version (7.5.12)
fhc --help       # lists every command
```

---

## 2. How the engine works (the mental model)

The engine is a set of **small commands that read and write plain files in the
current directory**. There is no database and no hidden state — everything the
game knows lives in the files in your game folder. That is exactly why git
works so well here: committing the folder snapshots the entire game.

The important files:

| File                         | What it is                                              |
|------------------------------|---------------------------------------------------------|
| `galaxy.dat`                 | The galaxy (size, turn number, species count)           |
| `stars.dat`, `planets.dat`   | The map                                                 |
| `sp01.dat` … `spNN.dat`      | One per species — its private game state                |
| `sp01.log` … `spNN.log`      | Per-species event log (accumulates across turns)        |
| `locations.dat`              | Where everything is this turn                           |
| `sp01.ord` … `spNN.ord`      | **Player orders** — the input you collect each turn     |
| `sp01.rpt.t1` … `spNN.rpt.tN`| **Player reports** — the output you mail back each turn  |
| `*.log` (combat.log, etc.)   | Per-phase processing logs — your troubleshooting trail  |

Species are numbered, and the number is zero-padded to two digits in every
filename. The report name is `sp<species>.rpt.t<turn>` — so species 1's orders
are `sp01.ord`, and its turn-3 report is `sp01.rpt.t3`.

### A turn is a pipeline of commands

Running a turn means running these commands **in order**, each reading the files
the previous one wrote:

```
locations → combat → pre-departure → jump → production → post-arrival → finish → report
```

You drop the players' `spNN.ord` files in before running `combat`; `report`
writes out the `spNN.rpt.tN` files at the end. `finish` advances the turn
number.

### The seed

The engine seeds its random number generator from the `FH_SEED` environment
variable. **Pick one number when you create the galaxy and keep it constant for
the entire life of the game** — re-running a turn with the same seed and same
orders produces identical results, which is what makes troubleshooting safe.

Put it in your shell profile or, better, a small file you `source` from the
game directory:

```sh
echo 'export FH_SEED=1924085713' > env.sh   # use your own number
source env.sh
```

Source it in every shell session you use to run the game.

---

## 3. Recommended directory structure

Keep the engine's working files **flat at the top of the game folder** (the
engine reads and writes the current directory), and use a couple of
subdirectories for your own bookkeeping:

```
acme-galaxy/              # the game folder == the git repository
├── env.sh                # export FH_SEED=...  (source before running)
├── species.cfg           # the species roster, used once at setup
├── galaxy.dat            # ─┐
├── stars.dat             #  │ engine working files — committed each turn
├── planets.dat           #  │
├── sp01.dat … spNN.dat   #  │
├── sp01.log … spNN.log   #  │
├── locations.dat         # ─┘
├── sp01.ord … spNN.ord   # orders you collect (committed as they arrive)
├── sp01.rpt.t1 …         # reports the engine writes (committed)
├── combat.log, jump.log… # phase logs (committed — your audit trail)
├── mail/
│   ├── in/               # raw order e-mails you saved, by turn
│   │   └── turn-03/
│   └── out/              # reports you actually sent, by turn
│       └── turn-03/
└── .gitignore
```

`mail/in` and `mail/out` are purely for your own records — copy raw incoming
e-mails into `mail/in/turn-NN/` and the reports you send into
`mail/out/turn-NN/`. They never affect the engine.

A reasonable `.gitignore`:

```gitignore
# the engine binary — rebuild it instead of committing it
fhc
```

Commit everything else. The `.dat` files, orders, reports, and logs are the
game; you *want* them in history.

---

## 4. Create the game folder and initialize git

```sh
mkdir acme-galaxy
cd acme-galaxy
git init
mkdir -p mail/in mail/out
printf 'fhc\n' > .gitignore
echo 'export FH_SEED=1924085713' > env.sh   # choose your own seed
source env.sh
git add .gitignore env.sh
git commit -m "Initialize Acme Galaxy game folder"
```

---

## 5. Set up a new game

Setup is three steps, all on `main`. Decide how many species (players) the game
will have first.

### 5a. Create the galaxy

```sh
fhc create galaxy --species=9
```

Useful options (run `fhc create galaxy --help` for the full list):

- `--species=N` — number of species the galaxy is sized for (**required**)
- `--stars=N` — number of stars (defaults to a value derived from `--species`)
- `--radius=N` — galactic radius in parsecs
- `--less-crowded` — spread things out more
- `--suggest-values` — print suggested star/radius values and exit (planning aid)

### 5b. Create the home-system templates

```sh
fhc create home-system-templates
```

### 5c. Define the species and create them

Write a `species.cfg` describing each player's species. The format is
`KEY VALUE`, one species per `species` block; indentation and blank lines are
ignored, `#` starts a comment. A minimal block:

```
species
    name      Alderaan
    homeworld Optimus
    govtname  His Majesty
    govttype  Degenerated Monarchy
    ml        10              # military level
    gv        1               # gravitics
    ls        1               # life support
    bi        3               # biology
    email     alderaan@example.com
```

(There is a fully worked `species.cfg` in the C engine's `examples/` folder you
can copy and adapt.) Then create them:

```sh
fhc create species --config=species.cfg --radius=6
```

`--radius` here is the minimum distance between home systems (must be ≤ half the
galactic radius).

### 5d. Commit the starting state

```sh
git add -A
git commit -m "Set up Acme Galaxy: galaxy, templates, 9 species"
git tag setup
```

At this point the game is at the start of **turn 1**. `fhc turn` will show the
current turn number; `fhc stats` prints a summary you can sanity-check.

---

## 6. Running a turn

This is the loop you repeat every turn. The idea: **one git branch per turn**.
You commit the orders as they arrive, commit the result after the pipeline runs
cleanly, and merge the branch back into `main` *without deleting it* so you (and
tools like GitKraken) can always find and review exactly what happened that
turn.

Throughout this section, substitute the real turn number for `NN` (e.g. `03`).

### 6a. Start the turn's branch

From `main`, with the previous turn already merged in:

```sh
source env.sh                 # make sure FH_SEED is set
git switch -c turn-NN         # e.g. git switch -c turn-03
```

### 6b. Collect orders as they come in

As each player's e-mail arrives:

1. Save the raw e-mail under `mail/in/turn-NN/` (your record).
2. Write their orders to the matching `spXX.ord` file at the top of the folder
   (species 4's orders go in `sp04.ord`).
3. Commit that one file so you have a per-player trail:

```sh
git add sp04.ord mail/in/turn-NN/
git commit -m "turn NN: orders from Alderaan (sp04)"
```

Players who don't send orders simply have no `spXX.ord` (or you can drop in the
"no orders received" note from the C engine's `examples/noorders.txt`). The
engine generates sensible default behavior for missing orders.

Once all the orders you're going to get are in and committed, you have a clean
**"orders placed" commit** — the point you'll come back to if a run goes wrong.

### 6c. Run the turn pipeline

Run the commands in order. Capturing each command's output to a `.log` file
gives you a record to troubleshoot from:

```sh
fhc locations      > locations.log    2>&1
fhc combat         > combat.log       2>&1
fhc pre-departure  > predeparture.log 2>&1
fhc jump           > jump.log         2>&1
fhc production     > production.log   2>&1
fhc post-arrival   > postarrival.log  2>&1
fhc finish         > finish.log       2>&1
fhc report         > report.log       2>&1
```

Then look things over:

```sh
fhc stats          # end-of-turn summary
fhc turn           # confirm the turn number advanced
```

You can also browse the galaxy with the read-only `list` and `show` commands
(`fhc list galaxy`, `fhc show galaxy --ascii`, etc.) — see the reference below.

### 6d. Troubleshoot and re-run (the safe part)

If a player's orders had a typo, or a phase produced something wrong, you do
**not** patch the half-processed `.dat` files. Instead, throw away the run,
fix the orders, and run the pipeline again from the clean "orders placed" state.
Because the run only changed tracked files and used a fixed `FH_SEED`, git makes
this trivial:

```sh
git restore .                 # discard the run; back to the orders-placed commit
# edit the offending spXX.ord, then commit the fix:
git add sp04.ord
git commit -m "turn NN: fix Alderaan's jump order (sp04)"
# re-run the whole pipeline from 6c
```

`git restore .` returns every tracked file to the last commit — which is your
orders-placed state — without touching your committed orders. Same seed + same
(corrected) orders = a clean, reproducible re-run. Repeat until the turn is
right.

> **Keep the whole history — never rewrite or squash it.** Each commit is
> evidence. Commit one order file per commit (6b) so the trail is precise. Then,
> if player 5 reports a problem, you can troubleshoot from the latest commit; if
> he insists he submitted an order you don't see, you can step back through the
> commit history to find when (and whether) it arrived; and if you uncover an
> engine bug, you can hand the maintainers the entire repository and they can
> reproduce the turn exactly. Squashing or amending commits throws all of that
> away. A few extra fix commits on a turn branch are not clutter — they're the
> record.

### 6e. Commit the finished turn

When the pipeline runs clean and the reports look right:

```sh
git add -A
git commit -m "turn NN: ran pipeline, reports generated"
```

### 6f. Mail the reports

The engine wrote one report per species, named `spXX.rpt.tNN`. Send each to the
corresponding player (their e-mail is in `species.cfg`). Keep copies of what you
actually sent:

```sh
mkdir -p mail/out/turn-NN
cp -p sp*.rpt.t<turn-number> mail/out/turn-NN/   # e.g. sp*.rpt.t3
git add mail/out/turn-NN/
git commit -m "turn NN: archived sent reports"
```

### 6g. Merge the turn back into main — without deleting the branch

This is the key habit. Merge with `--no-ff` so the merge always creates a
commit and the turn branch stays visible as its own line of history, and then
**leave the branch in place** so you (or GitKraken) can select it later to
review everything that happened that turn:

```sh
git switch main
git merge --no-ff turn-NN -m "Merge turn NN"
# do NOT run `git branch -d turn-NN` — keep the branch.
```

Optionally tag the end-of-turn state for quick navigation:

```sh
git tag turn-NN-done
```

That's the whole turn. Next turn: back to **6a** with `turn-$(NN+1)`.

---

## 7. A worked turn 1, start to finish

This section walks the whole loop once, concretely, for a small three-player
game. It assumes you've just finished setup (section 5) with three species —
**Alderaan (`sp01`)**, **Bantustan (`sp02`)**, and **Charabon (`sp03`)** — and
you're on `main`.

**Turn 1 is the startup turn.** Players have not received anything yet, so they
have nothing to write orders against. Running turn 1 is what *produces* their
first reports — each player's `sp0X.rpt.t1` shows them their home system and
starting position. So turn 1 normally runs with **default orders** (no player
input); real player orders start arriving for turn 2, written against these
turn-1 reports. (If you run a variant where players submit opening orders before
the game starts, drop those `spXX.ord` files in instead of generating defaults —
the rest of the steps are identical.)

First, confirm where you are. Right after setup the turn counter reads `0` — the
turn-1 pipeline's `finish` phase advances it to `1`:

```sh
$ source env.sh
$ fhc turn
0
```

### Start the turn-1 branch

```sh
$ git switch -c turn-01
Switched to a new branch 'turn-01'
```

### Lay down the default startup orders

For the startup turn, generate default orders for everyone and commit them so
the input is on the record just like any other turn:

```sh
$ fhc create orders > create_orders.log 2>&1
$ ls sp0?.ord
sp01.ord  sp02.ord  sp03.ord
$ git add sp01.ord sp02.ord sp03.ord create_orders.log
$ git commit -m "turn 01: default startup orders"
```

### Run the pipeline

```sh
$ fhc locations      > locations.log    2>&1
$ fhc combat         > combat.log       2>&1
$ fhc pre-departure  > predeparture.log 2>&1
$ fhc jump           > jump.log         2>&1
$ fhc production     > production.log   2>&1
$ fhc post-arrival   > postarrival.log  2>&1
$ fhc finish         > finish.log       2>&1
$ fhc report         > report.log       2>&1
```

### Verify the turn advanced and the reports were written

```sh
$ fhc turn
1
$ ls sp0?.rpt.t1
sp01.rpt.t1  sp02.rpt.t1  sp03.rpt.t1
$ fhc stats          # skim the end-of-turn summary
```

If anything looks wrong — a default you want to override, a phase that erred —
this is the moment to use the safe re-run from **6d**: `git restore .` back to
the "default startup orders" commit, adjust, and re-run the pipeline. Same
`FH_SEED`, same orders, identical result.

### Commit the finished turn

```sh
$ git add -A
$ git commit -m "turn 01: ran pipeline, reports generated"
```

### Mail the reports and archive what you sent

Send `sp01.rpt.t1` to Alderaan, `sp02.rpt.t1` to Bantustan, `sp03.rpt.t1` to
Charabon (their addresses are in `species.cfg`), then keep copies:

```sh
$ mkdir -p mail/out/turn-01
$ cp -p sp0?.rpt.t1 mail/out/turn-01/
$ git add mail/out/turn-01/
$ git commit -m "turn 01: archived sent reports"
```

### Merge back into main (keep the branch) and back up

```sh
$ git switch main
$ git merge --no-ff turn-01 -m "Merge turn 01"
$ git tag turn-01-done
$ git push origin --all --follow-tags     # off-machine backup (section 9)
```

Turn 1 is done. Now the players read their turn-1 reports and e-mail you orders
for turn 2 — and from here on you're in the steady-state loop of section 6:
`git switch -c turn-02`, commit each player's `spXX.ord` as it arrives, run the
pipeline, mail the `sp0X.rpt.t2` reports, merge, back up.

---

## 8. Reviewing history in GitKraken (or any git GUI)

Because you keep every `turn-NN` branch and merge with `--no-ff`, the commit
graph shows each turn as its own labeled branch joining back into `main`. To
review a turn:

- Select the `turn-NN` branch (or `turn-NN-done` tag) in the graph.
- Diff `turn-NN` against `main`'s previous merge — or just walk the branch's
  commits — to see exactly which orders came in, which were corrected, and what
  the pipeline changed in the `.dat` files and reports.
- The per-player order commits (6b) and the run commit (6c–6e) make it easy to
  answer "what did this player actually submit?" and "what changed when I
  processed the turn?" long after the fact.

---

## 9. Back up the repository

Your git folder *is* the game — the `.dat` files, every order, every report,
and the full history. If you lose it, you lose the game. Push it to a hosted
git service so there's always an off-machine copy, and so you can hand the whole
repository to the game's maintainers if you ever hit an engine bug.

> **Use a _private_ repository.** The folder contains your players' e-mail
> addresses (`species.cfg`, `mail/in/`) and their orders. None of that should be
> public. Every option below offers free private repos.

### Recommended: a private GitHub repo

GitHub is the natural choice — private repos are free, and the project itself
lives at `github.com/playbymail`, so sharing the repo with the maintainers is a
matter of adding them as collaborators. Create an **empty private** repo on
github.com (no README), then from your game folder:

```sh
git remote add origin git@github.com:YOUR-USERNAME/acme-galaxy.git
git push -u origin --all          # push main AND every turn-NN branch
git push origin --tags            # push the setup / turn-NN-done tags
```

The `--all` matters: a plain `git push` sends only the current branch, but you
deliberately keep every `turn-NN` branch, so push them all. After each turn
(once you've merged into `main` in step 6g), update the backup with:

```sh
git push origin --all --follow-tags
```

Run that as the last step of every turn and your off-machine copy is never more
than one turn behind. To give the maintainers access, add them as collaborators
in the repo's **Settings → Collaborators** (keeps it private), or for a one-off
bug report, create a `.zip` from GitHub's **Code → Download ZIP** or locally
with `git archive`.

### Alternatives

Any hosted git service works the same way — only the remote URL changes:

- **GitLab** (`gitlab.com`) and **Bitbucket** (`bitbucket.org`) — free private
  repos; identical `git remote add` / `git push --all` workflow.
- **Codeberg** (`codeberg.org`) — a nonprofit, no-tracking option, also free
  private repos.

### No-account option

You don't need a hosted service at all — you just need the copy to live
somewhere other than your machine. A **bare clone** on an external drive or in a
cloud-synced folder (iCloud Drive, Dropbox, Google Drive) is a complete backup:

```sh
# one-time: create the backup remote (e.g. on a USB drive)
git clone --bare . /Volumes/Backup/acme-galaxy.git
git remote add backup /Volumes/Backup/acme-galaxy.git

# after each turn:
git push backup --all --follow-tags
```

Putting that bare repo inside a cloud-synced folder gets you off-site backup
without any git host. You can keep both a GitHub remote *and* a drive/cloud
remote and push to each — more copies, more safety.

---

## 10. Command reference

Run any command with `--help` for its options. The ones you'll use for setup
and turns:

| Command                       | Purpose                                              |
|-------------------------------|------------------------------------------------------|
| `create galaxy`               | Create a new galaxy (start here)                     |
| `create home-system-templates`| Build the home-system templates                      |
| `create species --config=…`   | Create the player species from a `species.cfg`       |
| `locations`                   | First phase of a turn; update locations/efficiency   |
| `combat`                      | Resolve combat orders                                |
| `pre-departure`               | Pre-departure actions                                |
| `jump`                        | Resolve jumps / movement                             |
| `production`                  | Resolve production                                   |
| `post-arrival`                | Post-arrival actions                                 |
| `finish`                      | End-of-turn logic; advances the turn number          |
| `report`                      | Write the `spXX.rpt.tNN` player reports              |
| `turn`                        | Print the current turn number                        |
| `stats`                       | Print a galaxy/economy summary                       |

Read-only inspection (handy while running a turn, never changes state):

| Command                          | Purpose                                       |
|----------------------------------|-----------------------------------------------|
| `list galaxy`                    | List the galaxy (planets, optionally wormholes)|
| `list scanned --species=N`       | What species N has scanned                     |
| `show galaxy [--ascii]`          | Render the galaxy map (also writes `galaxy.map`)|
| `show <value> …`                 | Query game values (num_stars, turn_number, …)  |
| `scan` / `scan-near`             | Species-specific scan of a location            |
| `export json`                    | Dump the current state to JSON                 |
| `version`                        | Print the engine version                       |

`fhc create orders` exists too — it generates **default** orders for every
species. As a GM collecting real orders by e-mail you normally won't use it,
but it's there if you ever want to fill in defaults for everyone.

---

## 11. Quick checklist per turn

1. `source env.sh` and `git switch -c turn-NN`.
2. Save & commit each player's `spXX.ord` as it arrives (6b).
3. Run `locations → combat → pre-departure → jump → production → post-arrival → finish → report` (6c).
4. Check `stats` / `turn`; if wrong, `git restore .`, fix orders, re-run (6d).
5. `git add -A && git commit` the finished turn (6e).
6. Mail `spXX.rpt.tNN` to players; archive in `mail/out/turn-NN/` (6f).
7. `git switch main && git merge --no-ff turn-NN`; keep the branch; tag it (6g).
8. `git push origin --all --follow-tags` to back up the turn off-machine (§9).

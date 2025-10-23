# Game Master Manual for Far Horizons

This manual provides instructions for managing a Far Horizons game using the `fh` command-line tool.

## Setup

Before using the commands in this manual, ensure the `fh` executable is in your PATH or use the full path to the executable. Examples:

- Linux/macOS: `export PATH=$PATH:/path/to/fh/dist/local`
- Windows: `set PATH=%PATH%;C:\path\to\fh\dist\local`

Alternatively, you can use the full path to `fh` in each command (e.g., `/path/to/fh/dist/local/fh`).

## Create a new Game
Creating a game creates a new database file with the default values for a game.
It does not create a galaxy or add players.

To create a new game, run the following command:

```bash
fh init game --id gamma
```

The command accepts the following flags:

| flag  | meaning                           |          | default |
|-------|-----------------------------------|----------|---------|
| id    | identifier for the game           | required |         |
| path  | path to create the data files in  | optional | .       |
| force | overwrite any existing files      |          |         |

If the command completes successfully, you will have an initialized database. 

## Creating a New Galaxy

To create a new galaxy, follow these steps:

```bash
# Create a roomy galaxy with 18 potential homeworlds
fh create galaxy --less-crowded --species=18

# Show galaxy information
fh show galaxy

# Create home system templates
fh create home-system-templates

# Create species from configuration
fh create species --config=species.json

# Finish initial setup
fh run finish

# Generate initial ("Turn 0") reports
fh create turn-reports

# Display statistics
fh show stats
```

Notes:
1. You are not allowed to create multiple galaxies in the same game database.
2. Specify the `--path` parameter if you're not in the game's folder.

## Running a Turn

To process a complete game turn:

```bash
# Display current turn number
fh show turn

# Update locations and economic efficiency
fh run locations

# Process combat commands
fh run combat

# Execute pre-departure commands
fh run pre-departure

# Process jump commands
fh run jump

# Execute production commands
fh run production

# Execute post-arrival commands
fh run post-arrival

# Update locations again
fh run locations

# Process combat strikes
fh run combat --strike

# Finish turn processing
fh run finish

# Generate turn reports
fh create reports

# Display updated statistics
fh show stats
```

Notes:
1. Specify the `--path` parameter if you're not in the game's folder.

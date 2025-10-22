# Game Master Manual for Far Horizons

This manual provides instructions for managing a Far Horizons game using the `fh` command-line tool.

## Setup

Before using the commands in this manual, ensure the `fh` executable is in your PATH or use the full path to the executable. Examples:

- Linux/macOS: `export PATH=$PATH:/path/to/fh/dist/local`
- Windows: `set PATH=%PATH%;C:\path\to\fh\dist\local`

Alternatively, you can use the full path to `fh` in each command (e.g., `/path/to/fh/dist/local/fh`).

## Creating a New Galaxy

To create a new galaxy, follow these steps:

```bash
# Create a new directory for the game
mkdir gamma
cd gamma

# Copy initial configuration files (if available)
# cp examples/noorders.txt .
# cp examples/species.cfg .

# Create the galaxy with options
fh create galaxy --less-crowded --species=18

# Show galaxy information
fh show galaxy

# Create home system templates
fh create home-system-templates

# Create species from configuration
fh create species --config=species.cfg.json

# Finish initial setup
fh finish

# Generate initial reports
fh report

# Display statistics
fh stats
```

## Running a Turn

To process a complete game turn:

```bash
# Display current turn number
fh turn

# Update locations and economic efficiency
fh locations

# Process combat commands
fh combat

# Execute pre-departure commands
fh pre-departure

# Process jump commands
fh jump

# Execute production commands
fh production

# Execute post-arrival commands
fh post-arrival

# Update locations again
fh locations

# Process combat strikes
fh combat --strike

# Finish turn processing
fh finish

# Generate turn reports
fh report

# Display updated statistics
fh stats
```

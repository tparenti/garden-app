# garden

A CLI tool for tracking your seed inventory, planning when to plant things, and figuring out frost dates for your location.

Data is stored in a SQLite database at `~/.garden/garden.db`.

## Install

```
go build -o garden.exe .
```

Or just use `go run .` during development.

## First-time setup

Tell the app where you live so it can calculate frost dates:

```
garden locale set --zip 80203
# or by state:
garden locale set --state CO
```

## Commands

### seeds — your seed stash

```
garden seeds list
garden seeds add --name Tomato --variety "Cherokee Purple" --qty 2 --unit packets
garden seeds remove 3
garden seeds link <seed-id> <spec-id>   # connect a seed to a plant spec
```

### plants — the plant library

A built-in reference of plant specs with growing info (days to maturity, sun, spacing, frost timing, etc.).

```
garden plants list
garden plants list --sun full
garden plants search tomato
garden plants show 12
```

### schedule — plan what to plant and when

```
garden schedule list
garden schedule add --plant Tomato --type indoor_start --date 2026-03-01
garden schedule suggest --plant Tomato        # calculates dates from your frost dates
garden schedule done 4                        # mark entry #4 as planted
garden schedule remove 4
```

Planting types: `indoor_start`, `transplant`, `direct_sow`

The `suggest` command looks up your frost dates and tells you the optimal window to start seeds indoors or direct sow outside.

### locale — your location

```
garden locale show
garden locale set --zip 80203
garden locale set --state CO
```

### serve — web UI

```
garden serve
garden serve --port 9090
```

Opens a browser UI at `http://localhost:8080`.

## Database location

Default: `~/.garden/garden.db`

Override with the `--db` flag on any command:

```
garden --db ./mygarden.db seeds list
```

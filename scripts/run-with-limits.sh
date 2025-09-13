#!/usr/bin/env bash

# Run any command with conservative CPU/Memory/GC limits to avoid overloading WSL/host.
# Usage examples:
#   scripts/run-with-limits.sh golangci-lint run ./...
#   MAX_PROCS=1 MEM_LIMIT=800MiB scripts/run-with-limits.sh go vet ./...

set -euo pipefail

MAX_PROCS=${MAX_PROCS:-2}
MEM_LIMIT=${MEM_LIMIT:-1GiB}
GOGC_VALUE=${GOGC_VALUE:-50}

export GOMAXPROCS="$MAX_PROCS"
export GOMEMLIMIT="$MEM_LIMIT"
export GOGC="$GOGC_VALUE"

# Show usage if no arguments provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <command> [args...]"
    echo ""
    echo "Run any command with conservative CPU/Memory/GC limits to avoid overloading WSL/host."
    echo ""
    echo "Examples:"
    echo "  $0 golangci-lint run ./..."
    echo "  MAX_PROCS=1 MEM_LIMIT=800MiB $0 go vet ./..."
    echo ""
    echo "Environment variables (with defaults):"
    echo "  MAX_PROCS=$MAX_PROCS        - GOMAXPROCS setting"
    echo "  MEM_LIMIT=$MEM_LIMIT       - GOMEMLIMIT setting"
    echo "  GOGC_VALUE=$GOGC_VALUE           - GOGC setting"
    echo "  NICENESS=$NICENESS            - nice priority level"
    echo "  IONICENESS=$IONICENESS           - ionice priority level (if available)"
    exit 1
fi

# Lower scheduling priority to reduce host contention
NICENESS=${NICENESS:-10}
IONICENESS=${IONICENESS:-7}

echo "[run-with-limits] GOMAXPROCS=$GOMAXPROCS, GOMEMLIMIT=$GOMEMLIMIT, GOGC=$GOGC, nice=$NICENESS, ionice=$IONICENESS — executing: $*"
exec nice -n "$NICENESS" ionice -c2 -n "$IONICENESS" "$@"

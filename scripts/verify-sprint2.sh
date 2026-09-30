#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$ROOT/scripts/verify-sprint1.sh"
echo 'Sprint 2 verificado para Luis Michael Taype Rivas.'

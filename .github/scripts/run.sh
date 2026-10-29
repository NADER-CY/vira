#!/usr/bin/env bash
# Runs a command, streams its output, and on failure re-emits the last lines as GitHub error
# annotations so the failure is readable from the Checks API without downloading logs.
# usage: .github/scripts/run.sh "<title>" <command> [args...]
set -o pipefail
title="$1"; shift
log="$(mktemp)"
"$@" 2>&1 | tee "$log"
rc=${PIPESTATUS[0]}
if [ "$rc" -ne 0 ]; then
  tail -n 180 "$log" | sed 's/\x1b\[[0-9;]*[A-Za-z]//g' | split -l 30 - "$log.part."
  i=0
  for f in "$log".part.*; do
    i=$((i+1))
    msg=$(sed -e 's/%/%25/g' -e 's/\r/%0D/g' "$f" | awk 'BEGIN{ORS="%0A"} {print}')
    echo "::error title=${title} (${i})::${msg}"
  done
fi
exit "$rc"

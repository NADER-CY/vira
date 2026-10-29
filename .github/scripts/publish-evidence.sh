#!/usr/bin/env bash
# Publishes CI evidence (screenshots, generated lockfiles, test reports) to a throw-away branch
# ci-evidence/<name>-<run_id> so it can be reviewed and fetched with plain git. The branch is a single
# orphan commit; it never touches main or the feature branch. usage: publish-evidence.sh <name> <dir>
set -euo pipefail
name="$1"; dir="$2"
branch="ci-evidence/${name}-${GITHUB_RUN_ID}"
tmp="$(mktemp -d)"
cp -r "$dir"/. "$tmp"/
cd "$tmp"
git init -q -b "$branch"
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
printf 'CI evidence for %s\nrun: %s/%s/actions/runs/%s\ncommit: %s\n' "$name" "$GITHUB_SERVER_URL" "$GITHUB_REPOSITORY" "$GITHUB_RUN_ID" "$GITHUB_SHA" > README.txt
git add -A
git commit -qm "CI evidence: ${name} (run ${GITHUB_RUN_ID}, ${GITHUB_SHA})"
git push -q "https://x-access-token:${GH_TOKEN}@github.com/${GITHUB_REPOSITORY}.git" "HEAD:refs/heads/${branch}"
echo "::notice title=evidence::published ${branch}"

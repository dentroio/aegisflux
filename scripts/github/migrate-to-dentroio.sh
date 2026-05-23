#!/usr/bin/env bash
# Move sgerhart/aegisflux to dentroio/aegisflux and update the local git remote.
#
# Prerequisites:
#   - gh authenticated: gh auth status
#   - Admin on sgerhart/aegisflux
#   - Permission to create repositories in the dentroio org
#
# Usage:
#   ./scripts/github/migrate-to-dentroio.sh          # transfer + update remote
#   ./scripts/github/migrate-to-dentroio.sh --remote-only  # after a web UI transfer

set -euo pipefail

SOURCE_OWNER="sgerhart"
SOURCE_REPO="aegisflux"
TARGET_OWNER="dentroio"
TARGET_REPO="aegisflux"
REMOTE_ONLY=false

if [[ "${1:-}" == "--remote-only" ]]; then
  REMOTE_ONLY=true
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

require_gh() {
  if ! gh auth status >/dev/null 2>&1; then
    echo "gh is not authenticated. Run:"
    echo "  gh auth login -h github.com -p ssh -w"
    exit 1
  fi
}

transfer_repo() {
  require_gh

  if gh repo view "${TARGET_OWNER}/${TARGET_REPO}" >/dev/null 2>&1; then
    echo "Target repo ${TARGET_OWNER}/${TARGET_REPO} already exists."
    return 0
  fi

  echo "Transferring ${SOURCE_OWNER}/${SOURCE_REPO} -> ${TARGET_OWNER}/${TARGET_REPO}..."
  gh api "repos/${SOURCE_OWNER}/${SOURCE_REPO}/transfer" \
    -f new_owner="${TARGET_OWNER}" \
    -f new_name="${TARGET_REPO}"

  echo "Waiting for transfer to complete..."
  for _ in $(seq 1 30); do
    if gh repo view "${TARGET_OWNER}/${TARGET_REPO}" >/dev/null 2>&1; then
      echo "Transfer complete."
      return 0
    fi
    sleep 2
  done

  echo "Transfer initiated but ${TARGET_OWNER}/${TARGET_REPO} is not visible yet."
  echo "Check https://github.com/${TARGET_OWNER}/${TARGET_REPO} or org transfer approvals."
  exit 1
}

update_remote() {
  local new_url="git@github.com:${TARGET_OWNER}/${TARGET_REPO}.git"
  echo "Updating origin -> ${new_url}"
  git remote set-url origin "${new_url}"
  git remote -v
  git fetch origin
  echo "Remote updated and fetched successfully."
}

verify_ssh() {
  if ssh -o BatchMode=yes -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
    echo "SSH authentication to GitHub: OK"
  else
    echo "Warning: SSH authentication to GitHub may not be configured."
  fi
}

if [[ "$REMOTE_ONLY" == false ]]; then
  transfer_repo
fi

verify_ssh
update_remote

echo
echo "Done. Repository: https://github.com/${TARGET_OWNER}/${TARGET_REPO}"
echo "Next: push your branch with  git push -u origin main"

#!/usr/bin/env bash
# Configure dentroio/aegisflux branch protection and security features.
# Requires GitHub admin on the repository (or org owner).
#
# Usage:
#   gh auth refresh -h github.com -s admin:org,repo,read:org
#   ./scripts/github/setup-dentroio-repo.sh

set -euo pipefail

OWNER="dentroio"
REPO="aegisflux"
BRANCH="main"

require_admin() {
  local perm
  perm="$(gh api "repos/${OWNER}/${REPO}/collaborators/$(gh api user --jq .login)/permission" --jq .permission)"
  if [[ "$perm" != "admin" ]]; then
    echo "Current permission on ${OWNER}/${REPO}: ${perm}"
    echo "Admin access is required. Ask a dentroio org owner to:"
    echo "  1. Grant you admin on ${OWNER}/${REPO}, or"
    echo "  2. Run this script themselves after: gh auth login"
    exit 1
  fi
}

enable_security_features() {
  echo "Enabling Dependabot vulnerability alerts..."
  gh api -X PUT "repos/${OWNER}/${REPO}/vulnerability-alerts"

  echo "Enabling Dependabot security updates..."
  gh api -X PUT "repos/${OWNER}/${REPO}/automated-security-fixes"

  echo "Security features enabled."
}

protect_main_branch() {
  echo "Protecting branch ${BRANCH}..."
  gh api -X PUT "repos/${OWNER}/${REPO}/branches/${BRANCH}/protection" \
    --input - <<EOF
{
  "required_status_checks": {
    "strict": true,
    "checks": [
      {"context": "CI", "app_id": -1}
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false
  },
  "restrictions": null,
  "required_linear_history": false,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_conversation_resolution": true
}
EOF
  echo "Branch protection configured."
}

summarize_dependabot_prs() {
  echo
  echo "Open Dependabot pull requests:"
  gh pr list --repo "${OWNER}/${REPO}" --author app/dependabot --json number,title,mergeable,url \
    --jq '.[] | "#\(.number) mergeable=\(.mergeable) \(.title)\n  \(.url)"'
}

main() {
  require_admin
  enable_security_features
  protect_main_branch
  summarize_dependabot_prs
  echo
  echo "Done."
  echo "Review Dependabot PRs at: https://github.com/${OWNER}/${REPO}/pulls"
}

main "$@"

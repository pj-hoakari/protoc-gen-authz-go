#!/usr/bin/env bash
# Checks authz schema and plugin version bumps between two Git revisions.
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <base-revision> <head-revision>" >&2
  exit 2
fi

base_ref=$1
head_ref=$2
schema_path=proto/authz/v1/options.proto

die() {
  echo "check-authz-proto-version: $*" >&2
  exit 1
}

schema_version_at() {
  local revision=$1
  local content
  local -a versions

  content=$(git show "$revision:$schema_path") || die "cannot read $schema_path at $revision"
  mapfile -t versions < <(printf '%s\n' "$content" | sed -nE 's/^[[:space:]]*option[[:space:]]+\(authz\.v1\.authz_proto_version\)[[:space:]]*=[[:space:]]*([0-9]+)[[:space:]]*;[[:space:]]*$/\1/p')
  if [[ ${#versions[@]} -eq 0 ]]; then
    return 1
  fi
  [[ ${#versions[@]} -eq 1 ]] || die "$schema_path at $revision declares authz_proto_version more than once"
  [[ ${versions[0]} =~ ^[1-9][0-9]*$ ]] || die "$schema_path at $revision has an invalid authz_proto_version"
  printf '%s\n' "${versions[0]}"
}

plugin_version_in() {
  local directory=$1
  local version

  version=$(cd "$directory" && go run ./cmd/protoc-gen-authz-go --proto-version) || die "cannot obtain plugin version in $directory"
  [[ $version =~ ^[1-9][0-9]*$ ]] || die "plugin returned an invalid proto version: $version"
  printf '%s\n' "$version"
}

if git diff --quiet "$base_ref" "$head_ref" -- "$schema_path"; then
  exit 0
fi

head_schema_version=$(schema_version_at "$head_ref") || die "$schema_path changed, but head does not declare authz_proto_version"
head_plugin_version=$(plugin_version_in .)
[[ $head_schema_version == "$head_plugin_version" ]] || die "head version mismatch: authz_proto_version=$head_schema_version, plugin=$head_plugin_version"

if ! base_schema_version=$(schema_version_at "$base_ref"); then
  # The versioning feature is being introduced. There is no older value to
  # compare, but the head schema and plugin must already agree.
  exit 0
fi

temporary_directory=$(mktemp -d)
base_worktree=$temporary_directory/base
cleanup() {
  git worktree remove --force "$base_worktree" >/dev/null 2>&1 || true
  rm -rf "$temporary_directory"
}
trap cleanup EXIT

git worktree add --detach "$base_worktree" "$base_ref" >/dev/null || die "cannot create worktree for $base_ref"
base_plugin_version=$(plugin_version_in "$base_worktree")

if (( 10#$head_schema_version <= 10#$base_schema_version )); then
  die "$schema_path changed, but authz_proto_version was not incremented (base=$base_schema_version, head=$head_schema_version)"
fi
if (( 10#$head_plugin_version <= 10#$base_plugin_version )); then
  die "$schema_path changed, but plugin proto version was not incremented (base=$base_plugin_version, head=$head_plugin_version)"
fi

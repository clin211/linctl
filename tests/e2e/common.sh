#!/usr/bin/env bash
# Shared helpers for linctl E2E scripts.
# shellcheck shell=bash

# Scaffolded go.mod contains:
#   replace github.com/clin211/linhub => ../linhub
# so from $workspace_dir/myblog, ../linhub must exist. In E2E, $workspace_dir
# is a temp dir; symlink the real linhub repo there before go mod tidy / go build.
lin_e2e_link_linhub_replace() {
	local workspace_dir="$1"
	local lin_repo_root="$2"
	local linhub_src
	linhub_src="$(cd "$lin_repo_root/../linhub" && pwd)"
	if [ ! -f "$linhub_src/go.mod" ]; then
		echo "E2E: linhub not found at $linhub_src (expected sibling of lin repo)" >&2
		exit 1
	fi
	ln -sfn "$linhub_src" "$workspace_dir/linhub"
}

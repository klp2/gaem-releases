#!/usr/bin/env bash
# Read and compare-and-swap announcement state on the dedicated state branch.
set -euo pipefail

die() {
	printf 'announcement-state: %s\n' "$*" >&2
	exit 1
}

read_state() {
	local repo=$1 branch=$2 path=$3 head_out=$4 state_out=$5
	local owner name ref_response head expression response
	owner=${repo%%/*}
	name=${repo#*/}
	[[ -n "$owner" && -n "$name" && "$owner/$name" == "$repo" ]] ||
		die "repository must be owner/name"
	ref_response=$(gh api -H 'Cache-Control: no-cache' \
		"repos/$repo/git/ref/heads/$branch") ||
		die "cannot read announcement state branch"
	head=$(jq -er '.object.sha | select(test("^[0-9a-f]{40}$"))' <<<"$ref_response") ||
		die "announcement state branch is missing or malformed"
	expression="$head:$path"
	# Read through the immutable head: a moving branch expression can resolve
	# older contents than a separately resolved ref.
	# shellcheck disable=SC2016 # GraphQL variables are literal $ names.
	response=$(gh api graphql \
		-f query='query($owner:String!,$name:String!,$expression:String!){repository(owner:$owner,name:$name){object(expression:$expression){... on Blob{text}}}}' \
		-f owner="$owner" -f name="$name" -f expression="$expression") ||
		die "cannot read announcement state file"
	printf '%s\n' "$head" >"$head_out"
	# Stream blob text directly: command substitution strips trailing newlines,
	# while state transitions compare the exact canonical JSON bytes.
	jq -rj '.data.repository.object.text // ""' <<<"$response" >"$state_out" ||
		die "announcement state file response is malformed"
}

[[ $# -ge 1 ]] || die "usage: announcement-state.sh read|cas ..."
command=$1
shift

case "$command" in
read)
	[[ $# -eq 5 ]] || die "usage: announcement-state.sh read <repo> <branch> <path> <head-out> <state-out>"
	read_state "$@"
	;;
read-until-head)
	[[ $# -eq 6 ]] || die "usage: announcement-state.sh read-until-head <repo> <branch> <path> <expected-head> <head-out> <state-out>"
	repo=$1 branch=$2 path=$3 expected_head=$4 head_out=$5 state_out=$6
	[[ "$expected_head" =~ ^[0-9a-f]{40}$ ]] || die "expected head is malformed"
	for try in 1 2 3 4 5 6 7 8 9 10; do
		read_state "$repo" "$branch" "$path" "$head_out" "$state_out"
		[[ "$(cat "$head_out")" == "$expected_head" ]] && exit 0
		[[ "$try" == 10 ]] || sleep 1
	done
	die "announcement state branch did not converge to the expected head"
	;;
cas)
	[[ $# -eq 7 ]] || die "usage: announcement-state.sh cas <repo> <branch> <path> <expected-head> <state-file> <headline> <commit-out>"
	repo=$1 branch=$2 path=$3 expected_head=$4 state_file=$5 headline=$6 commit_out=$7
	[[ "$expected_head" =~ ^[0-9a-f]{40}$ ]] || die "expected head is malformed"
	[[ -s "$state_file" ]] || die "new state file is empty"
	contents=$(base64 -w0 <"$state_file")
	# shellcheck disable=SC2016 # GraphQL variables are literal $ names.
	response=$(gh api graphql \
		-f query='mutation($repo:String!,$branch:String!,$head:GitObjectID!,$headline:String!,$path:String!,$contents:Base64String!){createCommitOnBranch(input:{branch:{repositoryNameWithOwner:$repo,branchName:$branch},expectedHeadOid:$head,message:{headline:$headline},fileChanges:{additions:[{path:$path,contents:$contents}]}}){commit{oid}}}' \
		-f repo="$repo" -f branch="refs/heads/$branch" -f head="$expected_head" \
		-f headline="$headline" -f path="$path" -f contents="$contents") ||
		die "compare-and-swap commit failed"
	commit=$(jq -er '.data.createCommitOnBranch.commit.oid | select(test("^[0-9a-f]{40}$"))' <<<"$response") ||
		die "compare-and-swap response is malformed"
	printf '%s\n' "$commit" >"$commit_out"
	;;
*)
	die "unknown command $command"
	;;
esac

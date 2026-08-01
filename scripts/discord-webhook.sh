#!/usr/bin/env bash
set -euo pipefail
umask 077

die() {
  printf 'FATAL: Discord webhook operation failed (%s; details suppressed).\n' "$1" >&2
  exit 1
}

usage() {
  printf 'usage: %s check | send <message-file> <message-id-file>\n' "$0" >&2
  exit 2
}

mode="${1:-}"
case "$mode" in
  check)
    [[ "$#" -eq 1 ]] || usage
    ;;
  send)
    [[ "$#" -eq 3 ]] || usage
    message_file="$2"
    message_id_file="$3"
    [[ -s "$message_file" ]] || die "message is empty"
    ;;
  *)
    usage
    ;;
esac

[[ -n "${DISCORD_ANNOUNCE_WEBHOOK:-}" ]] || die "secret is not provisioned"
[[ -d "${RUNNER_TEMP:-}" ]] || die "runner temporary directory is unavailable"

scratch="$(mktemp -d "${RUNNER_TEMP%/}/discord-webhook.XXXXXX")"
trap 'rm -rf -- "$scratch"' EXIT
expected_channel="1525952505593462995"

check_identity() {
  local curl_rc status
  set +e
  status="$(curl --silent --output "$scratch/identity.json" \
    --write-out '%{http_code}' "$DISCORD_ANNOUNCE_WEBHOOK" \
    2>"$scratch/identity.err")"
  curl_rc=$?
  set -e
  [[ "$curl_rc" -eq 0 && "$status" == 200 ]] || die "identity check"
  jq -e --arg channel "$expected_channel" '.channel_id == $channel' \
    "$scratch/identity.json" >/dev/null || die "wrong channel"
}

check_identity
[[ "$mode" == send ]] || exit 0

character_count="$(wc -m <"$message_file")"
[[ "$character_count" =~ ^[0-9]+$ && "$character_count" -lt 2000 ]] ||
  die "message character limit"
if grep -Fq '```' "$message_file"; then
  die "message contains a code fence"
fi

jq -n --rawfile content "$message_file" \
  '{content:$content, allowed_mentions:{parse:[]}}' >"$scratch/payload.json"

set +e
status="$(curl --silent --output "$scratch/send.json" \
  --write-out '%{http_code}' -H 'Content-Type: application/json' \
  --request POST --data-binary @"$scratch/payload.json" \
  "${DISCORD_ANNOUNCE_WEBHOOK}?wait=true" 2>"$scratch/send.err")"
curl_rc=$?
set -e
[[ "$curl_rc" -eq 0 && "$status" == 200 ]] || die "send"

message_id="$(jq -er '.id | select(test("^[1-9][0-9]*$"))' "$scratch/send.json")" ||
  die "send response"

set +e
status="$(curl --silent --output "$scratch/readback.json" \
  --write-out '%{http_code}' \
  "${DISCORD_ANNOUNCE_WEBHOOK}/messages/$message_id" \
  2>"$scratch/readback.err")"
curl_rc=$?
set -e
[[ "$curl_rc" -eq 0 && "$status" == 200 ]] || die "readback"

jq -e --arg channel "$expected_channel" --rawfile expected "$message_file" \
  '.channel_id == $channel and .content == $expected' \
  "$scratch/readback.json" >/dev/null || die "readback mismatch"
if jq -r .content "$scratch/readback.json" | grep -Fq '```'; then
  die "readback contains a code fence"
fi

printf '%s\n' "$message_id" >"$message_id_file"

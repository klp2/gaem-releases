package announce

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAnnouncementWorkflowPinsAuthorityAndSecretHandling(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "announce.yml"))
	if err != nil {
		t.Fatal(err)
	}
	w := string(workflow)
	for _, want := range []string{
		"types: [published]",
		"workflow_dispatch:",
		"approved_message_sha256:",
		`case "$GITHUB_EVENT_NAME" in`,
		"release)",
		"workflow_dispatch)",
		"deliver:",
		"contents: read",
		"contents: write # expected-head commits on announcement-state only",
		"group: discord-announcement-state",
		"cancel-in-progress: false",
		"DISCORD_ANNOUNCE_WEBHOOK: ${{ secrets.DISCORD_ANNOUNCE_WEBHOOK }}",
		"go run ./cmd/announce plan",
		"go run ./cmd/announce state-plan",
		"go run ./cmd/announce reserve",
		"go run ./cmd/announce complete",
		"scripts/announcement-state.sh read-until-head",
		"announcements/v1/$RELEASE_ID.json",
		"probes/v1/discord-announcement.json",
		"go run ./cmd/announce probe-plan",
		"scripts/announcement-state.sh cas",
		"scripts/discord-webhook.sh check",
		"scripts/discord-webhook.sh send",
		`probe-plan "$probe_delivery_id"`,
		"cmp --silent",
	} {
		if !strings.Contains(w, want) {
			t.Errorf("workflow missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"set -x",
		"echo $DISCORD_ANNOUNCE_WEBHOOK",
		"printf $DISCORD_ANNOUNCE_WEBHOOK",
		"FEEDBACK_WEBHOOK",
		"RUNLOG_WEBHOOK",
		"pull-requests: write",
		"issues: write",
		"--method PATCH",
		"reserved-body.txt",
		"probe_message:",
		"  announce:\n",
		"  probe:\n",
	} {
		if strings.Contains(w, forbidden) {
			t.Errorf("workflow exposes secret or excess authority through %q", forbidden)
		}
	}
	inspect := strings.Index(w, "name: Inspect external idempotency state")
	preflight := strings.Index(w, "name: Verify announcements webhook channel")
	reserve := strings.Index(w, "name: Reserve this delivery attempt")
	post := strings.Index(w, "name: Send and read back the approved message")
	complete := strings.Index(w, "name: Record and verify the Discord message id")
	if inspect < 0 || preflight <= inspect || reserve <= preflight || post <= reserve || complete <= post {
		t.Errorf("workflow order does not enforce state inspect → preflight → CAS reserve → send/readback → CAS complete")
	}
	if strings.Count(w, "go run ./cmd/announce plan") != 2 {
		t.Error("workflow does not revalidate the full release immediately before send")
	}
	if strings.Count(w, "RELEASE_ID: ${{ github.event.release.id }}") != 2 {
		t.Error("initial and pre-send release planning do not both receive the event release id")
	}
	if strings.Count(w, "pre-send-plan.json") != 3 {
		t.Error("fresh pre-send plan output is not compared with the original plan")
	}
	if strings.Count(w, `cmp --silent "$reserved" "$RUNNER_TEMP/state.json"`) < 2 {
		t.Error("workflow does not bind send and ambiguous CAS recovery to the exact reserved state")
	}
	if strings.Count(w, "scripts/announcement-state.sh read-until-head") != 3 ||
		!strings.Contains(w, `reservation_head="$(cat "$RUNNER_TEMP/state-commit.txt")"`) ||
		!strings.Contains(w, `completion_head="$(cat "$RUNNER_TEMP/state-commit.txt")"`) {
		t.Error("workflow does not pin post-CAS reads to the exact returned commits")
	}
	if strings.Count(w, "go run ./cmd/announce probe-plan") != 2 {
		t.Error("probe plan is not regenerated immediately before send")
	}
	if strings.Count(w, `probe-plan "$probe_delivery_id"`) != 2 ||
		!strings.Contains(w, `probe_delivery_id="$GITHUB_RUN_ID"`) ||
		!strings.Contains(w, `.delivery_id | select(type == "number" and . > 0)`) {
		t.Error("probe recovery does not reuse the fixed state's delivery identity")
	}
	if strings.Count(w, `"$APPROVED_MESSAGE_SHA256"`) != 2 {
		t.Error("initial and pre-send probe plans are not bound to the approved digest")
	}
	if strings.Count(w, `state_path="probes/v1/discord-announcement.json"`) != 1 ||
		strings.Count(w, "STATE_PATH: ${{ steps.plan.outputs.state_path }}") != 3 {
		t.Error("delivery selection, inspect, reserve, and completion do not share the selected fixed state path")
	}
	if strings.Count(w, "scripts/discord-webhook.sh check") != 1 ||
		strings.Count(w, "scripts/discord-webhook.sh send") != 1 {
		t.Error("release and probe deliveries do not converge on one webhook check/send path")
	}
	approval := strings.Index(w, `probe-plan "$probe_delivery_id"`)
	probeSecret := strings.LastIndex(w, "DISCORD_ANNOUNCE_WEBHOOK: ${{ secrets.DISCORD_ANNOUNCE_WEBHOOK }}")
	if approval < 0 || probeSecret <= approval {
		t.Error("probe can access the webhook before binding the exact approved message digest")
	}
	if strings.Count(w, `if [ "$action" = fail ]`) != 1 {
		t.Error("the shared delivery path does not fail red on ambiguous sending state")
	}
	if !strings.Contains(w, `.state == "sent" and .attempt == $attempt and .message_id == $message_id`) {
		t.Error("probe does not read back and verify its final sent state")
	}

	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "discord-webhook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(script)
	for _, want := range []string{
		"1525952505593462995",
		"allowed_mentions:{parse:[]}",
		"?wait=true",
		"/messages/$message_id",
		".content == $expected",
		"details suppressed",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("webhook helper missing %q", want)
		}
	}
	for _, forbidden := range []string{"set -x", "echo $DISCORD_ANNOUNCE_WEBHOOK", "printf $DISCORD_ANNOUNCE_WEBHOOK"} {
		if strings.Contains(s, forbidden) {
			t.Errorf("webhook helper exposes secret through %q", forbidden)
		}
	}
}

func TestDiscordWebhookSendReadbackAndMentionSuppression(t *testing.T) {
	dir, command, env := discordWebhookFixture(t, "1525952505593462995", ProbeMessage)
	messageFile := filepath.Join(dir, "message.txt")
	messageIDFile := filepath.Join(dir, "message-id.txt")
	if err := os.WriteFile(messageFile, []byte(ProbeMessage), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(command, "send", messageFile, messageIDFile)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("send/readback failed: %v\n%s", err, out)
	}
	messageID, err := os.ReadFile(messageIDFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(messageID) != "1531120393325117462\n" {
		t.Fatalf("message id = %q", messageID)
	}
	payload, err := os.ReadFile(filepath.Join(dir, "payload.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Content         string `json:"content"`
		AllowedMentions struct {
			Parse []string `json:"parse"`
		} `json:"allowed_mentions"`
	}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got.Content != ProbeMessage || got.AllowedMentions.Parse == nil || len(got.AllowedMentions.Parse) != 0 {
		t.Fatalf("payload = %#v", got)
	}
	log, err := os.ReadFile(filepath.Join(dir, "curl.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(log), "POST") != 1 || !strings.Contains(string(log), "?wait=true") ||
		!strings.Contains(string(log), "/messages/1531120393325117462") {
		t.Fatalf("request log = %q", log)
	}
}

func TestDiscordWebhookFailsClosedWithoutDisclosingSecret(t *testing.T) {
	const secret = "https://discord.invalid/api/webhooks/42/do-not-print-this-token"
	tests := []struct {
		name, channel, readback, messageID string
		wantPost, wantReadback             bool
	}{
		{"wrong channel", "999", ProbeMessage, "1531120393325117462", false, false},
		{"malformed message id", "1525952505593462995", ProbeMessage, "not-a-snowflake", true, false},
		{"readback mismatch", "1525952505593462995", "changed", "1531120393325117462", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, command, env := discordWebhookFixture(t, tc.channel, tc.readback)
			env = replaceEnv(env, "DISCORD_ANNOUNCE_WEBHOOK", secret)
			env = replaceEnv(env, "FAKE_MESSAGE_ID", tc.messageID)
			messageFile := filepath.Join(dir, "message.txt")
			if err := os.WriteFile(messageFile, []byte(ProbeMessage), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(command, "send", messageFile, filepath.Join(dir, "message-id.txt"))
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("malformed webhook response survived")
			}
			if strings.Contains(string(out), secret) || strings.Contains(string(out), "do-not-print-this-token") {
				t.Fatalf("secret leaked in output: %s", out)
			}
			log, readErr := os.ReadFile(filepath.Join(dir, "curl.log"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			posted := strings.Contains(string(log), "POST")
			if posted != tc.wantPost {
				t.Fatalf("POST = %v, want %v; log = %q", posted, tc.wantPost, log)
			}
			readBack := strings.Contains(string(log), "/messages/")
			if readBack != tc.wantReadback {
				t.Fatalf("readback = %v, want %v; log = %q", readBack, tc.wantReadback, log)
			}
		})
	}
}

func TestDiscordWebhookRequiresSecretWithoutPrintingEnvironment(t *testing.T) {
	_, command, env := discordWebhookFixture(t, "1525952505593462995", ProbeMessage)
	env = replaceEnv(env, "DISCORD_ANNOUNCE_WEBHOOK", "")
	cmd := exec.Command(command, "check")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("missing webhook secret survived")
	}
	if strings.Contains(string(out), "discord.invalid") || strings.Contains(string(out), "test-token") {
		t.Fatalf("webhook environment leaked: %s", out)
	}
}

func TestDiscordWebhookRejectsInvalidMessageBeforePost(t *testing.T) {
	tests := []struct {
		name, message string
	}{
		{"code fence", ProbeMessage + "\n```"},
		{"character limit", strings.Repeat("é", 2_000)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, command, env := discordWebhookFixture(t, "1525952505593462995", tc.message)
			messageFile := filepath.Join(dir, "message.txt")
			if err := os.WriteFile(messageFile, []byte(tc.message), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(command, "send", messageFile, filepath.Join(dir, "message-id.txt"))
			cmd.Env = env
			if out, err := cmd.CombinedOutput(); err == nil {
				t.Fatalf("invalid message survived: %s", out)
			}
			log, err := os.ReadFile(filepath.Join(dir, "curl.log"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(log), "POST") {
				t.Fatalf("invalid message was posted: %q", log)
			}
		})
	}
}

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	replaced := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			replaced = append(replaced, entry)
		}
	}
	return append(replaced, prefix+value)
}

func discordWebhookFixture(t *testing.T, channel, readback string) (string, string, []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("webhook helper runs on ubuntu-latest")
	}
	for _, command := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s not on PATH", command)
		}
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	fakeCurl := filepath.Join(bin, "curl")
	const fake = `#!/usr/bin/env bash
set -euo pipefail
output=
method=GET
data=
url=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --silent) shift ;;
    --output|--write-out|-H|--request|--data-binary)
      key="$1"
      value="$2"
      shift 2
      case "$key" in
        --output) output="$value" ;;
        --request) method="$value" ;;
        --data-binary) data="$value" ;;
      esac
      ;;
    *) url="$1"; shift ;;
  esac
done
printf '%s\t%s\n' "$method" "$url" >>"$FAKE_CURL_LOG"
case "$method:$url" in
  POST:*)
    cp "${data#@}" "$FAKE_PAYLOAD"
    jq -n --arg id "$FAKE_MESSAGE_ID" '{id:$id}' >"$output"
    ;;
  GET:*/messages/1531120393325117462)
    jq -n --arg id '1531120393325117462' --arg channel "$FAKE_CHANNEL" \
      --rawfile content "$FAKE_READBACK" '{id:$id,channel_id:$channel,content:$content}' >"$output"
    ;;
  GET:*)
    jq -n --arg channel "$FAKE_CHANNEL" '{channel_id:$channel}' >"$output"
    ;;
  *) exit 9 ;;
esac
printf 200
`
	if err := os.WriteFile(fakeCurl, []byte(fake), 0o700); err != nil {
		t.Fatal(err)
	}
	readbackFile := filepath.Join(dir, "readback.txt")
	if err := os.WriteFile(readbackFile, []byte(readback), 0o600); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join("..", "..", "scripts", "discord-webhook.sh")
	env := append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"RUNNER_TEMP="+dir,
		"DISCORD_ANNOUNCE_WEBHOOK=https://discord.invalid/api/webhooks/42/test-token",
		"FAKE_CHANNEL="+channel,
		"FAKE_READBACK="+readbackFile,
		"FAKE_MESSAGE_ID=1531120393325117462",
		"FAKE_CURL_LOG="+filepath.Join(dir, "curl.log"),
		"FAKE_PAYLOAD="+filepath.Join(dir, "payload.json"),
	)
	return dir, command, env
}

func TestAnnouncementStateUsesExpectedHeadCAS(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "announcement-state.sh"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(script)
	for _, want := range []string{
		"createCommitOnBranch",
		"expectedHeadOid:$head",
		`branchName:$branch`,
		`branch="refs/heads/$branch"`,
		`repos/$repo/git/ref/heads/$branch`,
		"Cache-Control: no-cache",
		`expression="$head:$path"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("CAS helper missing %q", want)
		}
	}
	for _, forbidden := range []string{"--force", "PATCH", "DISCORD_ANNOUNCE_WEBHOOK", `expression="$branch:$path"`} {
		if strings.Contains(s, forbidden) {
			t.Errorf("CAS helper contains forbidden authority %q", forbidden)
		}
	}
}

func TestAnnouncementStateReadPreservesExactBlobBytes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("state helper runs on ubuntu-latest")
	}
	for _, command := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s not on PATH", command)
		}
	}
	dir := t.TempDir()
	fakeGH := filepath.Join(dir, "gh")
	const oid = "0123456789012345678901234567890123456789"
	response := `{"data":{"repository":{"object":{"text":"{\"state\":\"sending\"}\n"}}}}`
	fake := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$FAKE_GH_LOG"
case " $* " in
  *" repos/klp2/gaem-releases/git/ref/heads/announcement-state "*)
    printf '%s\n' '{"object":{"sha":"` + oid + `"}}'
    ;;
  *" expression=` + oid + `:announcements/v1/42.json "*)
    printf '%s\n' '` + response + `'
    ;;
  *) exit 9 ;;
esac
`
	if err := os.WriteFile(fakeGH, []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join("..", "..", "scripts", "announcement-state.sh")
	headFile := filepath.Join(dir, "head.txt")
	stateFile := filepath.Join(dir, "state.json")
	cmd := exec.Command("bash", script, "read", Repository, "announcement-state",
		"announcements/v1/42.json", headFile, stateFile)
	logFile := filepath.Join(dir, "gh.log")
	cmd.Env = append(os.Environ(),
		"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_GH_LOG="+logFile,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("state read failed: %v\n%s", err, out)
	}
	head, err := os.ReadFile(headFile)
	if err != nil {
		t.Fatal(err)
	}
	state, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(head) != oid+"\n" {
		t.Fatalf("head bytes = %q", head)
	}
	if string(state) != "{\"state\":\"sending\"}\n" {
		t.Fatalf("state bytes = %q; trailing newline was not preserved", state)
	}
	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	log := string(logBytes)
	refRead := strings.Index(log, "git/ref/heads/announcement-state")
	pinnedRead := strings.Index(log, "expression="+oid+":announcements/v1/42.json")
	if refRead < 0 || pinnedRead <= refRead || strings.Contains(log, "expression=announcement-state:") {
		t.Fatalf("state read was not pinned to the resolved head: %q", log)
	}
}

func TestAnnouncementStateReadWaitsForFreshExpectedHead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("state helper runs on ubuntu-latest")
	}
	for _, command := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s not on PATH", command)
		}
	}
	const (
		parent   = "1111111111111111111111111111111111111111"
		expected = "2222222222222222222222222222222222222222"
	)
	newFixture := func(t *testing.T, neverFresh bool) (string, []string) {
		t.Helper()
		dir := t.TempDir()
		fakeGH := filepath.Join(dir, "gh")
		fake := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$FAKE_GH_LOG"
case " $* " in
  *" repos/klp2/gaem-releases/git/ref/heads/announcement-state "*)
    count=0
    [ ! -f "$FAKE_REF_COUNT" ] || count=$(cat "$FAKE_REF_COUNT")
    count=$((count + 1))
    printf '%s\n' "$count" >"$FAKE_REF_COUNT"
    if [ "$FAKE_NEVER_FRESH" = true ] || [ "$count" -eq 1 ]; then
      printf '%s\n' '{"object":{"sha":"` + parent + `"}}'
    else
      printf '%s\n' '{"object":{"sha":"` + expected + `"}}'
    fi
    ;;
  *" expression=` + parent + `:announcements/v1/42.json "*)
    printf '%s\n' '{"data":{"repository":{"object":{"text":"{\"state\":\"pending\"}\n"}}}}'
    ;;
  *" expression=` + expected + `:announcements/v1/42.json "*)
    printf '%s\n' '{"data":{"repository":{"object":{"text":"{\"state\":\"sending\"}\n"}}}}'
    ;;
  *) exit 9 ;;
esac
`
		if err := os.WriteFile(fakeGH, []byte(fake), 0o755); err != nil {
			t.Fatal(err)
		}
		fakeSleep := filepath.Join(dir, "sleep")
		if err := os.WriteFile(fakeSleep, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		env := append(os.Environ(),
			"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
			"FAKE_GH_LOG="+filepath.Join(dir, "gh.log"),
			"FAKE_REF_COUNT="+filepath.Join(dir, "ref-count"),
			"FAKE_NEVER_FRESH="+map[bool]string{true: "true", false: "false"}[neverFresh],
		)
		return dir, env
	}

	t.Run("stale then fresh", func(t *testing.T) {
		dir, env := newFixture(t, false)
		headFile := filepath.Join(dir, "head.txt")
		stateFile := filepath.Join(dir, "state.json")
		cmd := exec.Command("bash", filepath.Join("..", "..", "scripts", "announcement-state.sh"),
			"read-until-head", Repository, "announcement-state", "announcements/v1/42.json",
			expected, headFile, stateFile)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fresh-head read failed: %v\n%s", err, out)
		}
		head, err := os.ReadFile(headFile)
		if err != nil {
			t.Fatal(err)
		}
		state, err := os.ReadFile(stateFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(head) != expected+"\n" || string(state) != "{\"state\":\"sending\"}\n" {
			t.Fatalf("fresh read = head %q state %q", head, state)
		}
		count, err := os.ReadFile(filepath.Join(dir, "ref-count"))
		if err != nil {
			t.Fatal(err)
		}
		if string(count) != "2\n" {
			t.Fatalf("ref reads = %q, want positive stale-then-fresh control", count)
		}
	})

	t.Run("never fresh", func(t *testing.T) {
		dir, env := newFixture(t, true)
		cmd := exec.Command("bash", filepath.Join("..", "..", "scripts", "announcement-state.sh"),
			"read-until-head", Repository, "announcement-state", "announcements/v1/42.json",
			expected, filepath.Join(dir, "head.txt"), filepath.Join(dir, "state.json"))
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "did not converge to the expected head") {
			t.Fatalf("non-converging read survived: %v\n%s", err, out)
		}
		count, err := os.ReadFile(filepath.Join(dir, "ref-count"))
		if err != nil {
			t.Fatal(err)
		}
		if string(count) != "10\n" {
			t.Fatalf("ref reads = %q, want bounded ten-attempt failure", count)
		}
	})
}

func TestPullRequestWorkflowRunsAllGoTests(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "test.yml"))
	if err != nil {
		t.Fatal(err)
	}
	w := string(workflow)
	for _, want := range []string{"pull_request:", "contents: read", "go-version-file: go.mod", "go test ./..."} {
		if !strings.Contains(w, want) {
			t.Errorf("test workflow missing %q", want)
		}
	}
}

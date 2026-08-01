package announce

import (
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
		"contents: read",
		"contents: write # expected-head commits on announcement-state only",
		"group: discord-announcement-state",
		"cancel-in-progress: false",
		"DISCORD_ANNOUNCE_WEBHOOK: ${{ secrets.DISCORD_ANNOUNCE_WEBHOOK }}",
		"1525952505593462995",
		"go run ./cmd/announce plan",
		"go run ./cmd/announce state-plan",
		"go run ./cmd/announce reserve",
		"go run ./cmd/announce complete",
		"announcements/v1/$RELEASE_ID.json",
		"scripts/announcement-state.sh cas",
		"allowed_mentions:{parse:[]}",
		"?wait=true",
		"/messages/$message_id",
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
	} {
		if strings.Contains(w, forbidden) {
			t.Errorf("workflow exposes secret or excess authority through %q", forbidden)
		}
	}
	inspect := strings.Index(w, "name: Inspect external idempotency state")
	preflight := strings.Index(w, "name: Verify announcements webhook channel")
	reserve := strings.Index(w, "name: Reserve this send attempt")
	post := strings.Index(w, "name: Send and read back the Discord message")
	complete := strings.Index(w, "name: Record the Discord message id")
	if inspect < 0 || preflight <= inspect || reserve <= preflight || post <= reserve || complete <= post {
		t.Errorf("workflow order does not enforce state inspect → preflight → CAS reserve → send/readback → CAS complete")
	}
	if strings.Count(w, "go run ./cmd/announce plan") != 2 {
		t.Error("workflow does not revalidate the full release immediately before send")
	}
	if strings.Count(w, "pre-send-plan.json") != 2 {
		t.Error("fresh pre-send plan output is not compared with the original plan")
	}
	if strings.Count(w, `cmp --silent "$reserved" "$RUNNER_TEMP/state.json"`) < 2 {
		t.Error("workflow does not bind send and ambiguous CAS recovery to the exact reserved state")
	}
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
	} {
		if !strings.Contains(s, want) {
			t.Errorf("CAS helper missing %q", want)
		}
	}
	for _, forbidden := range []string{"--force", "PATCH", "DISCORD_ANNOUNCE_WEBHOOK"} {
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
	response := `{"data":{"repository":{"ref":{"target":{"oid":"` + oid + `"}},"object":{"text":"{\"state\":\"sending\"}\n"}}}}`
	if err := os.WriteFile(fakeGH, []byte("#!/bin/sh\nprintf '%s\\n' '"+response+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join("..", "..", "scripts", "announcement-state.sh")
	headFile := filepath.Join(dir, "head.txt")
	stateFile := filepath.Join(dir, "state.json")
	cmd := exec.Command("bash", script, "read", Repository, "announcement-state",
		"announcements/v1/42.json", headFile, stateFile)
	cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
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

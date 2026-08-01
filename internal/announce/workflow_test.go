package announce

import (
	"os"
	"path/filepath"
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
		"contents: write # release-body idempotency state only",
		"cancel-in-progress: false",
		"DISCORD_ANNOUNCE_WEBHOOK: ${{ secrets.DISCORD_ANNOUNCE_WEBHOOK }}",
		"1525952505593462995",
		"go run ./cmd/announce plan",
		"go run ./cmd/announce reserve",
		"go run ./cmd/announce complete",
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
	} {
		if strings.Contains(w, forbidden) {
			t.Errorf("workflow exposes secret or excess authority through %q", forbidden)
		}
	}
	preflight := strings.Index(w, "name: Verify announcements webhook channel")
	reserve := strings.Index(w, "name: Reserve this send attempt")
	post := strings.Index(w, "name: Send and read back the Discord message")
	complete := strings.Index(w, "name: Record the Discord message id")
	if preflight < 0 || reserve <= preflight || post <= reserve || complete <= post {
		t.Errorf("workflow order does not enforce preflight → reserve → send/readback → complete")
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

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klp2/gaem-releases/internal/announce"
)

func TestProbePlanCommandBindsApprovedDigest(t *testing.T) {
	digestBytes := sha256.Sum256([]byte(announce.ProbeMessage))
	approved := hex.EncodeToString(digestBytes[:])
	out := filepath.Join(t.TempDir(), "plan.json")
	if err := run([]string{"probe-plan", "987654321", approved, out}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var plan announce.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.DeliveryID != 987654321 || plan.Message != announce.ProbeMessage {
		t.Fatalf("probe plan = %#v", plan)
	}

	for _, args := range [][]string{
		{"probe-plan", "0", approved, out},
		{"probe-plan", "not-a-run", approved, out},
		{"probe-plan", "987654321", strings.Repeat("0", 64), out},
		{"probe-plan", "987654321", approved},
	} {
		if err := run(args); err == nil {
			t.Fatalf("invalid probe command survived: %q", args)
		}
	}
}

func TestProbeMessageCommandWritesExactApprovalArtifacts(t *testing.T) {
	dir := t.TempDir()
	messagePath := filepath.Join(dir, "message.txt")
	digestPath := filepath.Join(dir, "sha256.txt")
	if err := run([]string{"probe-message", messagePath, digestPath}); err != nil {
		t.Fatal(err)
	}
	message, err := os.ReadFile(messagePath)
	if err != nil {
		t.Fatal(err)
	}
	digestValue, err := os.ReadFile(digestPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(message) != announce.ProbeMessage {
		t.Fatalf("message bytes = %q", message)
	}
	if string(digestValue) != announce.ProbeMessageDigest()+"\n" {
		t.Fatalf("digest bytes = %q", digestValue)
	}
	if err := run([]string{"probe-message", messagePath}); err == nil {
		t.Fatal("incomplete probe-message command survived")
	}
}

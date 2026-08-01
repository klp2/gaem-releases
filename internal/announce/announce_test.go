package announce

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func releaseFixture(tag string, prerelease bool, state, source, message string) Release {
	published := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	url := "https://github.com/klp2/gaem-releases/releases/tag/" + tag
	body := "source-sha: 0123456789012345678901234567890123456789\n\n" +
		StartMarker + "\n" +
		StatePrefix + state + "\n" +
		SourcePrefix + source + "\n\n" +
		message + "\n" + EndMarker
	return Release{ID: 42, TagName: tag, Prerelease: prerelease, HTMLURL: url, Body: body, PublishedAt: &published}
}

func messageFixture(tag string, rc bool, bullets ...string) string {
	headline := "## gaem " + tag + " is out"
	if rc {
		headline += " on the rc channel"
	}
	url := "https://github.com/klp2/gaem-releases/releases/tag/" + tag
	return headline + "\nGrab it here: <" + url + ">\n\n- " +
		strings.Join(bullets, "\n- ") + "\n\nFull notes on the release page."
}

func eventAndLiveJSON(t *testing.T, r Release) ([]byte, []byte) {
	t.Helper()
	e := map[string]any{
		"action":  "published",
		"release": r,
		"repository": map[string]any{
			"full_name": Repository,
		},
	}
	eventJSON, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	liveJSON, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return eventJSON, liveJSON
}

func planFor(t *testing.T, r Release) (Plan, error) {
	t.Helper()
	eventJSON, liveJSON := eventAndLiveJSON(t, r)
	return BuildPlan(eventJSON, liveJSON)
}

func TestBuildPlanAnnouncesStrictStableAndRC(t *testing.T) {
	tests := []struct {
		name, tag, source string
		prerelease        bool
		rc                bool
	}{
		{"stable", "v1.2.3", SourceStable, false, false},
		{"rc", "v1.3.0-rc.4", SourceRC, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			message := messageFixture(tc.tag, tc.rc, "One.", "Two.", "Three.")
			r := releaseFixture(tc.tag, tc.prerelease, "pending", tc.source, message)
			plan, err := planFor(t, r)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Action != ActionAnnounce || plan.Message != message || plan.Body != r.Body {
				t.Fatalf("plan = %#v", plan)
			}
		})
	}
}

func TestBuildPlanSkipsDraftNightlyDiagnosticsAndUnknownTags(t *testing.T) {
	tests := []struct {
		name, tag string
		draft     bool
	}{
		{"draft", "v1.2.3", true},
		{"nightly", "v1.3.0-nightly.20260731", false},
		{"diagnostic", "v1.3.0-diag1", false},
		{"unknown prerelease", "v1.3.0-beta.1", false},
		{"malformed stable", "v1.3", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := releaseFixture(tc.tag, strings.Contains(tc.tag, "-"), "pending", SourceRC, "unused")
			r.Draft = tc.draft
			plan, err := planFor(t, r)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Action != ActionSkip {
				t.Fatalf("plan action = %q, want skip", plan.Action)
			}
		})
	}
}

func TestBuildPlanFailsClosedOnMalformedRecognizedRelease(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Release)
		want string
	}{
		{"rc not prerelease", func(r *Release) { r.Prerelease = false }, "rc tag is not marked prerelease"},
		{"stable marked prerelease", func(r *Release) { r.Prerelease = true }, "stable tag is incorrectly marked prerelease"},
		{"missing publication", func(r *Release) { r.PublishedAt = nil }, "no published_at"},
		{"wrong source", func(r *Release) { r.Body = strings.Replace(r.Body, SourceRC, SourceStable, 1) }, "announcement source"},
		{"missing marker", func(r *Release) { r.Body = strings.Replace(r.Body, StartMarker, "", 1) }, "exactly one announcement marker pair"},
		{"duplicate marker", func(r *Release) { r.Body += "\n" + StartMarker }, "exactly one announcement marker pair"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tag := "v1.3.0-rc.1"
			prerelease := true
			source := SourceRC
			rc := true
			if strings.HasPrefix(tc.name, "stable") {
				tag, prerelease, source, rc = "v1.2.3", false, SourceStable, false
			}
			r := releaseFixture(tag, prerelease, "pending", source,
				messageFixture(tag, rc, "One.", "Two.", "Three."))
			tc.edit(&r)
			_, err := planFor(t, r)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestBuildPlanRejectsStaleOrMalformedAPIData(t *testing.T) {
	r := releaseFixture("v1.3.0-rc.1", true, "pending", SourceRC,
		messageFixture("v1.3.0-rc.1", true, "One.", "Two.", "Three."))
	eventJSON, liveJSON := eventAndLiveJSON(t, r)
	var live Release
	if err := json.Unmarshal(liveJSON, &live); err != nil {
		t.Fatal(err)
	}
	live.ID++
	changed, _ := json.Marshal(live)
	if _, err := BuildPlan(eventJSON, changed); err == nil || !strings.Contains(err.Error(), "fresh API read") {
		t.Fatalf("moving release survived: %v", err)
	}
	if _, err := BuildPlan(eventJSON, []byte(`{"id":`)); err == nil || !strings.Contains(err.Error(), "decode live release") {
		t.Fatalf("malformed live JSON survived: %v", err)
	}
}

func TestMessageValidation(t *testing.T) {
	tag := "v1.3.0-rc.1"
	valid := messageFixture(tag, true, "One.", "Two.", "Three.")
	tests := []struct {
		name, message, want string
	}{
		{"two bullets", messageFixture(tag, true, "One.", "Two."), "expected 3–5"},
		{"six bullets", messageFixture(tag, true, "One.", "Two.", "Three.", "Four.", "Five.", "Six."), "expected 3–5"},
		{"missing link", strings.Replace(valid, "<https://", "https://", 1), "exactly one release link"},
		{"fenced", valid + "\n```", "code fence"},
		{"wrong headline", strings.Replace(valid, "rc channel", "nightly channel", 1), "headline"},
		{"missing footer", strings.TrimSpace(strings.TrimSuffix(valid, "Full notes on the release page.")), "full-notes footer"},
		{"too long", strings.Replace(valid, "One.", strings.Repeat("é", 2_000), 1), "fewer than 2,000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := releaseFixture(tag, true, "pending", SourceRC, tc.message)
			_, err := planFor(t, r)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}

	unicodeMessage := strings.Replace(valid, "One.", strings.Repeat("é", 1_000), 1)
	if _, err := planFor(t, releaseFixture(tag, true, "pending", SourceRC, unicodeMessage)); err != nil {
		t.Fatalf("Unicode message below the rune cap failed: %v", err)
	}
}

func TestStateMachinePreventsDuplicateSendOnRetry(t *testing.T) {
	tag := "v1.3.0-rc.1"
	r := releaseFixture(tag, true, "pending", SourceRC,
		messageFixture(tag, true, "One.", "Two.", "Three."))
	reserved, err := Reserve(r.Body, "123:1")
	if err != nil {
		t.Fatal(err)
	}
	r.Body = reserved
	plan, err := planFor(t, r)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != ActionSkip || !strings.Contains(plan.Reason, "ambiguous") {
		t.Fatalf("reserved retry plan = %#v", plan)
	}
	if _, err := Reserve(reserved, "123:2"); err == nil {
		t.Fatal("second reservation survived an existing sending state")
	}
	if !strings.Contains(reserved, "sending:123:1:"+messageDigest(messageFixture(tag, true, "One.", "Two.", "Three."))) {
		t.Fatalf("reserved body does not bind its message digest:\n%s", reserved)
	}

	completed, err := Complete(reserved, "123:1", "1531120393325117462")
	if err != nil {
		t.Fatal(err)
	}
	r.Body = completed
	plan, err = planFor(t, r)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != ActionSkip || !strings.Contains(plan.Reason, "already sent") {
		t.Fatalf("completed retry plan = %#v", plan)
	}
	if !strings.Contains(completed, "sent:1531120393325117462") ||
		strings.Contains(completed, "sending:123:1") {
		t.Fatalf("completed body has wrong state:\n%s", completed)
	}
}

func TestReservedStateRejectsMessageMutationAndSupportsApprovedReset(t *testing.T) {
	tag := "v1.3.0-rc.1"
	r := releaseFixture(tag, true, "pending", SourceRC,
		messageFixture(tag, true, "One.", "Two.", "Three."))
	reserved, err := Reserve(r.Body, "123:1")
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(reserved, "- One.", "- Changed after reservation.", 1)
	r.Body = mutated
	if _, err := planFor(t, r); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("mutated reserved message survived planning: %v", err)
	}
	if _, err := Complete(mutated, "123:1", "999"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mutated reserved message survived completion: %v", err)
	}
	if _, err := Reset(mutated, "123:1"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mutated reserved message survived reset: %v", err)
	}
	if _, err := Reset(reserved, "123:2"); err == nil {
		t.Fatal("reset accepted a different run attempt")
	}
	pending, err := Reset(reserved, "123:1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pending, StatePrefix+"pending") || strings.Contains(pending, "sending:123:1") {
		t.Fatalf("approved reset produced wrong state:\n%s", pending)
	}
}

func TestStateTransitionsRejectMalformedIdentifiersAndWrongAttempts(t *testing.T) {
	r := releaseFixture("v1.3.0-rc.1", true, "pending", SourceRC,
		messageFixture("v1.3.0-rc.1", true, "One.", "Two.", "Three."))
	for _, attempt := range []string{"", "0:1", "1:0", "one:1", "1"} {
		if _, err := Reserve(r.Body, attempt); err == nil {
			t.Errorf("Reserve accepted attempt %q", attempt)
		}
	}
	reserved, err := Reserve(r.Body, "12:3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Complete(reserved, "12:4", "123"); err == nil {
		t.Fatal("Complete accepted a different run attempt")
	}
	if _, err := Complete(reserved, "12:3", "not-a-snowflake"); err == nil {
		t.Fatal("Complete accepted a malformed Discord message id")
	}
}

func TestEnvelopeParserDoesNotConfuseMessageContentForContainer(t *testing.T) {
	tag := "v1.3.0-rc.1"
	message := messageFixture(tag, true, "One.", "Two.", "Three.") +
		fmt.Sprintf("\n%s fake", StatePrefix)
	r := releaseFixture(tag, true, "pending", SourceRC, message)
	_, err := planFor(t, r)
	if err == nil || !strings.Contains(err.Error(), "exactly one announcement state") {
		t.Fatalf("container-confusing message survived: %v", err)
	}
}

func TestEnvelopeRejectsOuterMessageWhitespace(t *testing.T) {
	tag := "v1.2.3"
	message := messageFixture(tag, false, "One.", "Two.", "Three.")
	r := releaseFixture(tag, false, "pending", SourceStable, message)
	if _, err := parseEnvelope(r.Body); err != nil {
		t.Fatalf("positive control failed to parse: %v", err)
	}
	r.Body = strings.Replace(r.Body, message+"\n"+EndMarker, message+"\n\n"+EndMarker, 1)
	_, err := parseEnvelope(r.Body)
	if err == nil || !strings.Contains(err.Error(), "outer whitespace") {
		t.Fatalf("newline-terminated stable payload survived: %v", err)
	}
}

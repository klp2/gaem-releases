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

// untaggedDraftURL is the html_url GitHub serves for a draft. Measured
// 2026-08-03 by creating a draft here and reading it back, with and without the
// git tag already present — the untagged form holds either way.
//
// Pairing Draft with the permanent tag URL is a combination the API cannot
// return, which is what let this gate pass while rejecting every real draft.
// Mutating releaseLink back to release.HTMLURL reds
// TestValidateDraftUsesConsumerContract.
const untaggedDraftURL = "https://github.com/klp2/gaem-releases/releases/tag/untagged-759142ade0b365798f48"

func asDraft(r Release) Release {
	r.Draft = true
	r.HTMLURL = untaggedDraftURL
	return r
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
			r := releaseFixture(tc.tag, tc.prerelease, "external", tc.source, message)
			plan, err := planFor(t, r)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Action != ActionAnnounce || plan.Message != message || plan.Body != r.Body {
				t.Fatalf("plan = %#v", plan)
			}
			if plan.DeliveryID != r.ID || plan.Subject != r.TagName {
				t.Fatalf("release identity was not preserved as delivery identity: %#v", plan)
			}
		})
	}
}

func TestBuildProbePlanUsesFixedApprovedMessage(t *testing.T) {
	approved := digest(ProbeMessage)
	plan, err := BuildProbePlan(987654321, approved)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action != ActionAnnounce || plan.DeliveryID != 987654321 ||
		plan.Subject != ProbeSubject || plan.Source != SourceProbe || plan.Body != ProbeBody ||
		plan.URL != LatestReleaseURL || plan.Message != ProbeMessage {
		t.Fatalf("probe plan = %#v", plan)
	}
	if err := validateProbeMessage(plan.Message); err != nil {
		t.Fatalf("fixed probe message failed its own contract: %v", err)
	}
	if _, err := BuildProbePlan(0, approved); err == nil {
		t.Fatal("non-positive probe run id survived")
	}
	for _, badDigest := range []string{"", strings.ToUpper(approved), strings.Repeat("0", 64)} {
		if _, err := BuildProbePlan(987654321, badDigest); err == nil {
			t.Fatalf("unapproved probe digest %q survived", badDigest)
		}
	}

	mutations := []struct {
		name, message, want string
	}{
		{"headline", strings.Replace(ProbeMessage, "rollout probe", "test", 1), "headline"},
		{"two bullets", strings.Replace(ProbeMessage, "\n- It confirms exact message readback and duplicate prevention.", "", 1), "expected 3"},
		{"empty bullet", strings.Replace(ProbeMessage, "- It confirms exact message", "- \nIt confirms exact message", 1), "empty bullet"},
		{"link", strings.Replace(ProbeMessage, "<"+LatestReleaseURL+">", LatestReleaseURL, 1), "latest-release link"},
		{"fence", ProbeMessage + "\n```", "forbidden text"},
		{"caveat", strings.TrimSuffix(ProbeMessage, "Probe only — no player action needed."), "caveat"},
		{"length", ProbeMessage + strings.Repeat("é", 2_000), "fewer than 2,000"},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProbeMessage(tc.message)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestValidateDraftUsesConsumerContract(t *testing.T) {
	tag := "v1.3.0-rc.4"
	message := messageFixture(tag, true, "One.", "Two.", "Three.")
	release := asDraft(releaseFixture(tag, true, "external", SourceRC, message))
	_, releaseJSON := eventAndLiveJSON(t, release)
	if err := ValidateDraft(releaseJSON); err != nil {
		t.Fatalf("valid draft failed: %v", err)
	}

	tests := []struct {
		name string
		edit func(*Release)
		want string
	}{
		{"published", func(r *Release) { r.Draft = false }, "not a draft"},
		{"announcement links the draft's own untagged url", func(r *Release) {
			r.Body = strings.Replace(r.Body, tagURLPrefix+tag, untaggedDraftURL, 1)
		}, "exactly one release link"},
		{"unknown tag", func(r *Release) { r.TagName = "v1.3.0-beta.1" }, "not a stable or rc"},
		{"wrong source", func(r *Release) { r.Body = strings.Replace(r.Body, SourceRC, SourceStable, 1) }, "announcement source"},
		{"body-local state", func(r *Release) { r.Body = strings.Replace(r.Body, StatePrefix+"external", StatePrefix+"pending", 1) }, "is not external"},
		{"duplicate end marker", func(r *Release) { r.Body = strings.Replace(r.Body, "- One.", "- One. "+EndMarker, 1) }, "exactly one announcement marker pair"},
		{"state text in message", func(r *Release) { r.Body = strings.Replace(r.Body, "- One.", "- One. "+StatePrefix+"pending", 1) }, "exactly one announcement state and source"},
		{"short message", func(r *Release) {
			r.Body = strings.Replace(r.Body, message, messageFixture(tag, true, "One.", "Two."), 1)
		}, "expected 3–5"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mutated := release
			tc.edit(&mutated)
			_, gotJSON := eventAndLiveJSON(t, mutated)
			err := ValidateDraft(gotJSON)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
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
			r := releaseFixture(tc.tag, strings.Contains(tc.tag, "-"), "external", SourceRC, "unused")
			if tc.draft {
				r = asDraft(r)
			}
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
			r := releaseFixture(tag, prerelease, "external", source,
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
	r := releaseFixture("v1.3.0-rc.1", true, "external", SourceRC,
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
			r := releaseFixture(tag, true, "external", SourceRC, tc.message)
			_, err := planFor(t, r)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}

	unicodeMessage := strings.Replace(valid, "One.", strings.Repeat("é", 1_000), 1)
	if _, err := planFor(t, releaseFixture(tag, true, "external", SourceRC, unicodeMessage)); err != nil {
		t.Fatalf("Unicode message below the rune cap failed: %v", err)
	}
}

func TestStateMachinePreventsDuplicateSendOnRetry(t *testing.T) {
	tag := "v1.3.0-rc.1"
	r := releaseFixture(tag, true, "external", SourceRC,
		messageFixture(tag, true, "One.", "Two.", "Three."))
	plan, err := planFor(t, r)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := ReserveState(plan, nil, "123:1")
	if err != nil {
		t.Fatal(err)
	}
	reservedJSON, _ := json.Marshal(reserved)
	decision, err := InspectState(plan, reservedJSON)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionFail || !strings.Contains(decision.Reason, "ambiguous") {
		t.Fatalf("reserved retry decision = %#v", decision)
	}
	if _, err := ReserveState(plan, reservedJSON, "123:2"); err == nil {
		t.Fatal("second reservation survived an existing sending state")
	}
	if reserved.MessageSHA256 != digest(plan.Message) || reserved.BodySHA256 != digest(plan.Body) {
		t.Fatalf("reserved state does not bind its payload: %#v", reserved)
	}

	completed, err := CompleteState(plan, reservedJSON, "123:1", "1531120393325117462")
	if err != nil {
		t.Fatal(err)
	}
	completedJSON, _ := json.Marshal(completed)
	decision, err = InspectState(plan, completedJSON)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionSkip || !strings.Contains(decision.Reason, "already sent") {
		t.Fatalf("completed retry decision = %#v", decision)
	}
	if completed.State != "sent" || completed.MessageID != "1531120393325117462" || completed.Generation != 2 {
		t.Fatalf("completed external state is wrong: %#v", completed)
	}
}

func TestProbeStateRejectsASecondDispatch(t *testing.T) {
	approved := digest(ProbeMessage)
	first, err := BuildProbePlan(1001, approved)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := ReserveState(first, nil, "1001:1")
	if err != nil {
		t.Fatal(err)
	}
	completed, err := CompleteState(first, mustJSON(t, reserved), "1001:1", "1531120393325117462")
	if err != nil {
		t.Fatal(err)
	}
	decision, err := InspectState(first, mustJSON(t, completed))
	if err != nil || decision.Action != ActionSkip || !strings.Contains(decision.Reason, "already sent") {
		t.Fatalf("same-run retry decision = %#v, %v", decision, err)
	}
	second, err := BuildProbePlan(1002, approved)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InspectState(second, mustJSON(t, completed)); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("second dispatch survived fixed probe state: %v", err)
	}
	stateJSON := string(mustJSON(t, completed))
	if !strings.Contains(stateJSON, `"delivery_id":1001`) || strings.Contains(stateJSON, `"release_id"`) {
		t.Fatalf("probe state does not use generic delivery identity: %s", stateJSON)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestReservedStateRejectsMessageMutationAndSupportsApprovedReset(t *testing.T) {
	tag := "v1.3.0-rc.1"
	r := releaseFixture(tag, true, "external", SourceRC,
		messageFixture(tag, true, "One.", "Two.", "Three."))
	plan, err := planFor(t, r)
	if err != nil {
		t.Fatal(err)
	}
	reserved, err := ReserveState(plan, nil, "123:1")
	if err != nil {
		t.Fatal(err)
	}
	reservedJSON, _ := json.Marshal(reserved)
	bodyMutatedPlan := plan
	bodyMutatedPlan.Body = strings.Replace(plan.Body, "source-sha: 0123", "source-sha: 9999", 1)
	if _, err := InspectState(bodyMutatedPlan, reservedJSON); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("mutated release body survived state planning: %v", err)
	}
	messageMutatedPlan := plan
	messageMutatedPlan.Message = strings.Replace(plan.Message, "- One.", "- Changed after reservation.", 1)
	if _, err := InspectState(messageMutatedPlan, reservedJSON); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("mutated message survived state planning: %v", err)
	}
	if _, err := CompleteState(messageMutatedPlan, reservedJSON, "123:1", "999"); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("mutated reserved message survived completion: %v", err)
	}
	if _, err := ResetState(messageMutatedPlan, reservedJSON, "123:1"); err == nil || !strings.Contains(err.Error(), "does not bind") {
		t.Fatalf("mutated reserved message survived reset: %v", err)
	}
	if _, err := ResetState(plan, reservedJSON, "123:2"); err == nil {
		t.Fatal("reset accepted a different run attempt")
	}
	pending, err := ResetState(plan, reservedJSON, "123:1")
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != "pending" || pending.Attempt != "" || pending.Generation != 2 {
		t.Fatalf("approved reset produced wrong state: %#v", pending)
	}
	pendingJSON, _ := json.Marshal(pending)
	retry, err := ReserveState(plan, pendingJSON, "124:1")
	if err != nil || retry.State != "sending" || retry.Generation != 3 {
		t.Fatalf("approved retry reservation = %#v, %v", retry, err)
	}
}

func TestStateTransitionsRejectMalformedIdentifiersAndWrongAttempts(t *testing.T) {
	r := releaseFixture("v1.3.0-rc.1", true, "external", SourceRC,
		messageFixture("v1.3.0-rc.1", true, "One.", "Two.", "Three."))
	plan, err := planFor(t, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []string{"", "0:1", "1:0", "one:1", "1"} {
		if _, err := ReserveState(plan, nil, attempt); err == nil {
			t.Errorf("Reserve accepted attempt %q", attempt)
		}
	}
	reserved, err := ReserveState(plan, nil, "12:3")
	if err != nil {
		t.Fatal(err)
	}
	reservedJSON, _ := json.Marshal(reserved)
	if _, err := CompleteState(plan, reservedJSON, "12:4", "123"); err == nil {
		t.Fatal("Complete accepted a different run attempt")
	}
	if _, err := CompleteState(plan, reservedJSON, "12:3", "not-a-snowflake"); err == nil {
		t.Fatal("Complete accepted a malformed Discord message id")
	}
	withUnknown := strings.TrimSuffix(string(reservedJSON), "}") + `,"unexpected":true}`
	if _, err := InspectState(plan, []byte(withUnknown)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("state with unknown field survived: %v", err)
	}
}

func TestEnvelopeParserDoesNotConfuseMessageContentForContainer(t *testing.T) {
	tag := "v1.3.0-rc.1"
	message := messageFixture(tag, true, "One.", "Two.", "Three.") +
		fmt.Sprintf("\n%s fake", StatePrefix)
	r := releaseFixture(tag, true, "external", SourceRC, message)
	_, err := planFor(t, r)
	if err == nil || !strings.Contains(err.Error(), "exactly one announcement state") {
		t.Fatalf("container-confusing message survived: %v", err)
	}
}

func TestEnvelopeRejectsOuterMessageWhitespace(t *testing.T) {
	tag := "v1.2.3"
	message := messageFixture(tag, false, "One.", "Two.", "Three.")
	r := releaseFixture(tag, false, "external", SourceStable, message)
	if _, err := parseEnvelope(r.Body); err != nil {
		t.Fatalf("positive control failed to parse: %v", err)
	}
	r.Body = strings.Replace(r.Body, message+"\n"+EndMarker, message+"\n\n"+EndMarker, 1)
	_, err := parseEnvelope(r.Body)
	if err == nil || !strings.Contains(err.Error(), "outer whitespace") {
		t.Fatalf("newline-terminated stable payload survived: %v", err)
	}
}

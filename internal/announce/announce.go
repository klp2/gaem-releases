package announce

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	Repository     = "klp2/gaem-releases"
	StartMarker    = "<!-- gaem-discord-announcement:v1:start -->"
	EndMarker      = "<!-- gaem-discord-announcement:v1:end -->"
	StatePrefix    = "discord-announcement-state: "
	SourcePrefix   = "discord-announcement-source: "
	SourceRC       = "changelog-nightly"
	SourceStable   = "changelog-public-approved"
	ActionAnnounce = "announce"
	ActionSkip     = "skip"
)

var (
	stableTag   = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	rcTag       = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)-rc\.[1-9][0-9]*$`)
	nightlyTag  = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)-nightly\.[0-9]{8}$`)
	attemptID   = regexp.MustCompile(`^[1-9][0-9]*:[1-9][0-9]*$`)
	messageID   = regexp.MustCompile(`^[1-9][0-9]*$`)
	digestValue = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Release struct {
	ID          int64      `json:"id"`
	TagName     string     `json:"tag_name"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	HTMLURL     string     `json:"html_url"`
	Body        string     `json:"body"`
	PublishedAt *time.Time `json:"published_at"`
}

type event struct {
	Action     string  `json:"action"`
	Release    Release `json:"release"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type Plan struct {
	Action    string `json:"action"`
	Reason    string `json:"reason"`
	ReleaseID int64  `json:"release_id"`
	TagName   string `json:"tag_name"`
	URL       string `json:"url"`
	Source    string `json:"source,omitempty"`
	Body      string `json:"body,omitempty"`
	Message   string `json:"message,omitempty"`
}

type State struct {
	Version       int    `json:"version"`
	Generation    int    `json:"generation"`
	ReleaseID     int64  `json:"release_id"`
	TagName       string `json:"tag_name"`
	Source        string `json:"source"`
	BodySHA256    string `json:"body_sha256"`
	MessageSHA256 string `json:"message_sha256"`
	State         string `json:"state"`
	Attempt       string `json:"attempt,omitempty"`
	MessageID     string `json:"message_id,omitempty"`
}

type StatePlan struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type envelope struct {
	state, source, message string
}

// ValidateDraft validates the exact announcement contract embedded in a fresh
// draft release. Producers run this command from the deployed consumer source
// before undrafting, so release.published cannot expose an envelope that the
// consumer will reject.
func ValidateDraft(releaseJSON []byte) error {
	var release Release
	if err := decodeOne(releaseJSON, &release); err != nil {
		return fmt.Errorf("decode draft release: %w", err)
	}
	if release.ID <= 0 || release.TagName == "" || release.HTMLURL == "" {
		return errors.New("draft release is missing id, tag, or html_url")
	}
	if !release.Draft {
		return errors.New("release is not a draft")
	}
	channel, expectedSource, recognized, err := classifyRelease(release)
	if err != nil {
		return err
	}
	if !recognized {
		return fmt.Errorf("draft tag %q is not a stable or rc release", release.TagName)
	}
	return validateEnvelope(release, channel, expectedSource)
}

func BuildPlan(eventJSON, liveJSON []byte) (Plan, error) {
	var e event
	if err := decodeOne(eventJSON, &e); err != nil {
		return Plan{}, fmt.Errorf("decode release event: %w", err)
	}
	var live Release
	if err := decodeOne(liveJSON, &live); err != nil {
		return Plan{}, fmt.Errorf("decode live release: %w", err)
	}
	if e.Action != "published" {
		return Plan{}, fmt.Errorf("unexpected release action %q", e.Action)
	}
	if e.Repository.FullName != Repository {
		return Plan{}, fmt.Errorf("event repository %q is not %s", e.Repository.FullName, Repository)
	}
	if live.ID <= 0 || live.TagName == "" || live.HTMLURL == "" {
		return Plan{}, errors.New("live release is missing id, tag, or html_url")
	}
	if e.Release.ID != live.ID || e.Release.TagName != live.TagName ||
		e.Release.Draft != live.Draft || e.Release.Prerelease != live.Prerelease ||
		e.Release.HTMLURL != live.HTMLURL {
		return Plan{}, errors.New("release event disagrees with the fresh API read")
	}
	base := Plan{ReleaseID: live.ID, TagName: live.TagName, URL: live.HTMLURL}
	if live.Draft {
		base.Action, base.Reason = ActionSkip, "draft release"
		return base, nil
	}

	channel, expectedSource, recognized, err := classifyRelease(live)
	if err != nil {
		return Plan{}, err
	}
	if !recognized {
		switch {
		case nightlyTag.MatchString(live.TagName):
			base.Action, base.Reason = ActionSkip, "nightly release"
			return base, nil
		default:
			base.Action, base.Reason = ActionSkip, "diagnostic or unknown release tag"
			return base, nil
		}
	}
	if live.PublishedAt == nil {
		return Plan{}, errors.New("announced release has no published_at timestamp")
	}

	if err := validateEnvelope(live, channel, expectedSource); err != nil {
		return Plan{}, err
	}
	env, err := parseEnvelope(live.Body)
	if err != nil {
		return Plan{}, err
	}
	base.Source, base.Body, base.Message = env.source, live.Body, env.message
	base.Action, base.Reason = ActionAnnounce, "validated announcement with external CAS state"
	return base, nil
}

func classifyRelease(release Release) (channel, expectedSource string, recognized bool, err error) {
	switch {
	case stableTag.MatchString(release.TagName):
		if release.Prerelease {
			return "", "", true, errors.New("stable tag is incorrectly marked prerelease")
		}
		return "stable", SourceStable, true, nil
	case rcTag.MatchString(release.TagName):
		if !release.Prerelease {
			return "", "", true, errors.New("rc tag is not marked prerelease")
		}
		return "rc", SourceRC, true, nil
	default:
		return "", "", false, nil
	}
}

func validateEnvelope(release Release, channel, expectedSource string) error {
	env, err := parseEnvelope(release.Body)
	if err != nil {
		return err
	}
	if env.state != "external" {
		return fmt.Errorf("announcement envelope state %q is not external", env.state)
	}
	if env.source != expectedSource {
		return fmt.Errorf("%s release carries announcement source %q, want %q", channel, env.source, expectedSource)
	}
	return validateMessage(env.message, release.TagName, release.HTMLURL, channel)
}

func InspectState(plan Plan, stateJSON []byte) (StatePlan, error) {
	if err := validateAnnouncePlan(plan); err != nil {
		return StatePlan{}, err
	}
	if len(strings.TrimSpace(string(stateJSON))) == 0 {
		return StatePlan{Action: "reserve", Reason: "no prior state"}, nil
	}
	state, err := decodeState(stateJSON)
	if err != nil {
		return StatePlan{}, err
	}
	if err := stateMatchesPlan(state, plan); err != nil {
		return StatePlan{}, err
	}
	switch state.State {
	case "pending":
		if state.Attempt != "" || state.MessageID != "" {
			return StatePlan{}, errors.New("pending state carries attempt or message id")
		}
		return StatePlan{Action: "reserve", Reason: "approved recovery reset"}, nil
	case "sending":
		if !attemptID.MatchString(state.Attempt) || state.MessageID != "" {
			return StatePlan{}, errors.New("sending state is malformed")
		}
		return StatePlan{Action: ActionSkip, Reason: "prior send attempt is ambiguous; manual recovery required"}, nil
	case "sent":
		if !attemptID.MatchString(state.Attempt) || !messageID.MatchString(state.MessageID) {
			return StatePlan{}, errors.New("sent state is malformed")
		}
		return StatePlan{Action: ActionSkip, Reason: "announcement already sent"}, nil
	default:
		return StatePlan{}, fmt.Errorf("unknown external announcement state %q", state.State)
	}
}

func ReserveState(plan Plan, stateJSON []byte, attempt string) (State, error) {
	if !attemptID.MatchString(attempt) {
		return State{}, fmt.Errorf("invalid attempt id %q", attempt)
	}
	decision, err := InspectState(plan, stateJSON)
	if err != nil {
		return State{}, err
	}
	if decision.Action != "reserve" {
		return State{}, fmt.Errorf("cannot reserve: %s", decision.Reason)
	}
	generation := 1
	if len(strings.TrimSpace(string(stateJSON))) != 0 {
		prior, err := decodeState(stateJSON)
		if err != nil {
			return State{}, err
		}
		generation = prior.Generation + 1
	}
	return newState(plan, generation, "sending", attempt, ""), nil
}

func CompleteState(plan Plan, stateJSON []byte, attempt, discordMessageID string) (State, error) {
	if !attemptID.MatchString(attempt) {
		return State{}, fmt.Errorf("invalid attempt id %q", attempt)
	}
	if !messageID.MatchString(discordMessageID) {
		return State{}, fmt.Errorf("invalid Discord message id %q", discordMessageID)
	}
	state, err := decodeState(stateJSON)
	if err != nil {
		return State{}, err
	}
	if err := stateMatchesPlan(state, plan); err != nil {
		return State{}, err
	}
	if state.State != "sending" || state.Attempt != attempt || state.MessageID != "" {
		return State{}, fmt.Errorf("cannot complete state %q for attempt %q", state.State, attempt)
	}
	return newState(plan, state.Generation+1, "sent", attempt, discordMessageID), nil
}

func ResetState(plan Plan, stateJSON []byte, attempt string) (State, error) {
	if !attemptID.MatchString(attempt) {
		return State{}, fmt.Errorf("invalid attempt id %q", attempt)
	}
	state, err := decodeState(stateJSON)
	if err != nil {
		return State{}, err
	}
	if err := stateMatchesPlan(state, plan); err != nil {
		return State{}, err
	}
	if state.State != "sending" || state.Attempt != attempt || state.MessageID != "" {
		return State{}, fmt.Errorf("cannot reset state %q for attempt %q", state.State, attempt)
	}
	return newState(plan, state.Generation+1, "pending", "", ""), nil
}

func decodeState(data []byte) (State, error) {
	var state State
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decode announcement state: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return State{}, errors.New("decode announcement state: multiple JSON values")
		}
		return State{}, fmt.Errorf("decode announcement state: %w", err)
	}
	if state.Version != 1 || state.Generation < 1 || state.ReleaseID <= 0 || state.TagName == "" ||
		state.Source == "" || !digestValue.MatchString(state.BodySHA256) ||
		!digestValue.MatchString(state.MessageSHA256) {
		return State{}, errors.New("announcement state metadata is malformed")
	}
	return state, nil
}

func validateAnnouncePlan(plan Plan) error {
	if plan.Action != ActionAnnounce || plan.ReleaseID <= 0 || plan.TagName == "" ||
		plan.Source == "" || plan.Body == "" || plan.Message == "" {
		return errors.New("announcement plan is incomplete or not announceable")
	}
	return nil
}

func stateMatchesPlan(state State, plan Plan) error {
	if err := validateAnnouncePlan(plan); err != nil {
		return err
	}
	if state.ReleaseID != plan.ReleaseID || state.TagName != plan.TagName || state.Source != plan.Source ||
		state.BodySHA256 != digest(plan.Body) || state.MessageSHA256 != digest(plan.Message) {
		return errors.New("external announcement state does not bind the current release and message")
	}
	return nil
}

func newState(plan Plan, generation int, state, attempt, messageID string) State {
	return State{Version: 1, Generation: generation, ReleaseID: plan.ReleaseID, TagName: plan.TagName,
		Source: plan.Source, BodySHA256: digest(plan.Body), MessageSHA256: digest(plan.Message),
		State: state, Attempt: attempt, MessageID: messageID}
}

func decodeOne(data []byte, dst any) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func parseEnvelope(body string) (envelope, error) {
	if strings.Contains(body, "\r") {
		return envelope{}, errors.New("release body contains carriage returns")
	}
	if strings.Count(body, StartMarker) != 1 || strings.Count(body, EndMarker) != 1 {
		return envelope{}, errors.New("release body must carry exactly one announcement marker pair")
	}
	if strings.Count(body, StatePrefix) != 1 || strings.Count(body, SourcePrefix) != 1 {
		return envelope{}, errors.New("release body must carry exactly one announcement state and source")
	}
	start := strings.Index(body, StartMarker) + len(StartMarker)
	end := strings.Index(body, EndMarker)
	if start >= end || !strings.HasPrefix(body[start:end], "\n") || !strings.HasSuffix(body[start:end], "\n") {
		return envelope{}, errors.New("announcement markers are out of order or not line-delimited")
	}
	block := strings.TrimSuffix(strings.TrimPrefix(body[start:end], "\n"), "\n")
	lines := strings.Split(block, "\n")
	if len(lines) < 5 || !strings.HasPrefix(lines[0], StatePrefix) ||
		!strings.HasPrefix(lines[1], SourcePrefix) || lines[2] != "" {
		return envelope{}, errors.New("announcement envelope header is malformed")
	}
	message := strings.Join(lines[3:], "\n")
	if message == "" || strings.TrimSpace(message) != message {
		return envelope{}, errors.New("announcement message is empty or has outer whitespace")
	}
	return envelope{
		state:   strings.TrimPrefix(lines[0], StatePrefix),
		source:  strings.TrimPrefix(lines[1], SourcePrefix),
		message: message,
	}, nil
}

func validateMessage(message, tag, releaseURL, channel string) error {
	count := utf8.RuneCountInString(message)
	if count >= 2000 {
		return fmt.Errorf("announcement is %d characters; must be fewer than 2,000", count)
	}
	if strings.Contains(message, "```") {
		return errors.New("announcement contains a code fence")
	}
	if strings.Contains(message, StartMarker) || strings.Contains(message, EndMarker) || strings.Contains(message, "\r") {
		return errors.New("announcement contains forbidden envelope text")
	}
	lines := strings.Split(message, "\n")
	wantHeadline := "## gaem " + tag + " is out"
	if channel == "rc" {
		wantHeadline += " on the rc channel"
	}
	if len(lines) == 0 || lines[0] != wantHeadline {
		return fmt.Errorf("announcement headline = %q, want %q", lines[0], wantHeadline)
	}
	bullets := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "- ") {
			if strings.TrimSpace(strings.TrimPrefix(line, "- ")) == "" {
				return errors.New("announcement contains an empty bullet")
			}
			bullets++
		}
	}
	if bullets < 3 || bullets > 5 {
		return fmt.Errorf("announcement has %d bullets; expected 3–5", bullets)
	}
	link := "<" + releaseURL + ">"
	if strings.Count(message, link) != 1 {
		return fmt.Errorf("announcement must carry exactly one release link %s", link)
	}
	if !strings.HasSuffix(message, "Full notes on the release page.") {
		return errors.New("announcement is missing the full-notes footer")
	}
	return nil
}

func digest(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

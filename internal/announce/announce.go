package announce

import (
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
	stableTag  = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	rcTag      = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)-rc\.[1-9][0-9]*$`)
	nightlyTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)-nightly\.[0-9]{8}$`)
	attemptID  = regexp.MustCompile(`^[1-9][0-9]*:[1-9][0-9]*$`)
	messageID  = regexp.MustCompile(`^[1-9][0-9]*$`)
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
	Body      string `json:"body,omitempty"`
	Message   string `json:"message,omitempty"`
}

type envelope struct {
	state, source, message string
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

	channel := ""
	expectedSource := ""
	switch {
	case stableTag.MatchString(live.TagName):
		channel, expectedSource = "stable", SourceStable
		if live.Prerelease {
			return Plan{}, errors.New("stable tag is incorrectly marked prerelease")
		}
	case rcTag.MatchString(live.TagName):
		channel, expectedSource = "rc", SourceRC
		if !live.Prerelease {
			return Plan{}, errors.New("rc tag is not marked prerelease")
		}
	case nightlyTag.MatchString(live.TagName):
		base.Action, base.Reason = ActionSkip, "nightly release"
		return base, nil
	default:
		base.Action, base.Reason = ActionSkip, "diagnostic or unknown release tag"
		return base, nil
	}
	if live.PublishedAt == nil {
		return Plan{}, errors.New("announced release has no published_at timestamp")
	}

	env, err := parseEnvelope(live.Body)
	if err != nil {
		return Plan{}, err
	}
	if env.source != expectedSource {
		return Plan{}, fmt.Errorf("%s release carries announcement source %q, want %q", channel, env.source, expectedSource)
	}
	if err := validateMessage(env.message, live.TagName, live.HTMLURL, channel); err != nil {
		return Plan{}, err
	}
	base.Body, base.Message = live.Body, env.message
	switch {
	case env.state == "pending":
		base.Action, base.Reason = ActionAnnounce, "validated pending announcement"
	case strings.HasPrefix(env.state, "sending:"):
		if !attemptID.MatchString(strings.TrimPrefix(env.state, "sending:")) {
			return Plan{}, fmt.Errorf("malformed sending state %q", env.state)
		}
		base.Action, base.Reason = ActionSkip, "prior send attempt is ambiguous; manual recovery required"
	case strings.HasPrefix(env.state, "sent:"):
		if !messageID.MatchString(strings.TrimPrefix(env.state, "sent:")) {
			return Plan{}, fmt.Errorf("malformed sent state %q", env.state)
		}
		base.Action, base.Reason = ActionSkip, "announcement already sent"
	default:
		return Plan{}, fmt.Errorf("unknown announcement state %q", env.state)
	}
	return base, nil
}

func Reserve(body, attempt string) (string, error) {
	if !attemptID.MatchString(attempt) {
		return "", fmt.Errorf("invalid attempt id %q", attempt)
	}
	env, err := parseEnvelope(body)
	if err != nil {
		return "", err
	}
	if env.state != "pending" {
		return "", fmt.Errorf("cannot reserve announcement in state %q", env.state)
	}
	return replaceState(body, "pending", "sending:"+attempt)
}

func Complete(body, attempt, discordMessageID string) (string, error) {
	if !attemptID.MatchString(attempt) {
		return "", fmt.Errorf("invalid attempt id %q", attempt)
	}
	if !messageID.MatchString(discordMessageID) {
		return "", fmt.Errorf("invalid Discord message id %q", discordMessageID)
	}
	env, err := parseEnvelope(body)
	if err != nil {
		return "", err
	}
	want := "sending:" + attempt
	if env.state != want {
		return "", fmt.Errorf("cannot complete announcement in state %q; want %q", env.state, want)
	}
	return replaceState(body, want, "sent:"+discordMessageID)
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

func replaceState(body, from, to string) (string, error) {
	oldLine := StatePrefix + from
	if strings.Count(body, oldLine) != 1 {
		return "", fmt.Errorf("state line %q is not unique", oldLine)
	}
	return strings.Replace(body, oldLine, StatePrefix+to, 1), nil
}

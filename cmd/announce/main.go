package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/klp2/gaem-releases/internal/announce"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "announce:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: announce validate-draft|plan|probe-message|probe-plan|state-plan|reserve|complete|reset ...")
	}
	switch args[0] {
	case "validate-draft":
		if len(args) != 2 {
			return fmt.Errorf("usage: announce validate-draft <release.json>")
		}
		releaseJSON, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		return announce.ValidateDraft(releaseJSON)
	case "plan":
		if len(args) != 4 {
			return fmt.Errorf("usage: announce plan <event.json> <release.json> <plan.json>")
		}
		eventJSON, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		liveJSON, err := os.ReadFile(args[2])
		if err != nil {
			return err
		}
		plan, err := announce.BuildPlan(eventJSON, liveJSON)
		if err != nil {
			return err
		}
		out, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(args[3], append(out, '\n'), 0o600)
	case "probe-plan":
		if len(args) != 4 {
			return fmt.Errorf("usage: announce probe-plan <run-id> <approved-message-sha256> <plan.json>")
		}
		runID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parse probe run id: %w", err)
		}
		plan, err := announce.BuildProbePlan(runID, args[2])
		if err != nil {
			return err
		}
		return writeJSON(args[3], plan)
	case "probe-message":
		if len(args) != 3 {
			return fmt.Errorf("usage: announce probe-message <message.txt> <sha256.txt>")
		}
		if err := os.WriteFile(args[1], []byte(announce.ProbeMessage), 0o600); err != nil {
			return err
		}
		return os.WriteFile(args[2], []byte(announce.ProbeMessageDigest()+"\n"), 0o600)
	case "reserve":
		if len(args) != 5 {
			return fmt.Errorf("usage: announce reserve <plan.json> <state.json> <run:attempt> <reserved.json>")
		}
		plan, err := readPlan(args[1])
		if err != nil {
			return err
		}
		stateJSON, err := os.ReadFile(args[2])
		if err != nil {
			return err
		}
		reserved, err := announce.ReserveState(plan, stateJSON, args[3])
		if err != nil {
			return err
		}
		return writeJSON(args[4], reserved)
	case "state-plan":
		if len(args) != 4 {
			return fmt.Errorf("usage: announce state-plan <plan.json> <state.json> <decision.json>")
		}
		plan, err := readPlan(args[1])
		if err != nil {
			return err
		}
		stateJSON, err := os.ReadFile(args[2])
		if err != nil {
			return err
		}
		decision, err := announce.InspectState(plan, stateJSON)
		if err != nil {
			return err
		}
		return writeJSON(args[3], decision)
	case "complete":
		if len(args) != 6 {
			return fmt.Errorf("usage: announce complete <plan.json> <state.json> <run:attempt> <message-id> <complete.json>")
		}
		plan, err := readPlan(args[1])
		if err != nil {
			return err
		}
		stateJSON, err := os.ReadFile(args[2])
		if err != nil {
			return err
		}
		complete, err := announce.CompleteState(plan, stateJSON, args[3], args[4])
		if err != nil {
			return err
		}
		return writeJSON(args[5], complete)
	case "reset":
		if len(args) != 5 {
			return fmt.Errorf("usage: announce reset <plan.json> <state.json> <run:attempt> <pending.json>")
		}
		plan, err := readPlan(args[1])
		if err != nil {
			return err
		}
		stateJSON, err := os.ReadFile(args[2])
		if err != nil {
			return err
		}
		pending, err := announce.ResetState(plan, stateJSON, args[3])
		if err != nil {
			return err
		}
		return writeJSON(args[4], pending)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func readPlan(path string) (announce.Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return announce.Plan{}, err
	}
	var plan announce.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return announce.Plan{}, err
	}
	return plan, nil
}

func writeJSON(path string, value any) error {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}

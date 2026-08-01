package main

import (
	"encoding/json"
	"fmt"
	"os"

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
		return fmt.Errorf("usage: announce plan|reserve|complete ...")
	}
	switch args[0] {
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
	case "reserve":
		if len(args) != 4 {
			return fmt.Errorf("usage: announce reserve <body.txt> <run:attempt> <reserved.txt>")
		}
		body, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		reserved, err := announce.Reserve(string(body), args[2])
		if err != nil {
			return err
		}
		return os.WriteFile(args[3], []byte(reserved), 0o600)
	case "complete":
		if len(args) != 5 {
			return fmt.Errorf("usage: announce complete <body.txt> <run:attempt> <message-id> <complete.txt>")
		}
		body, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		complete, err := announce.Complete(string(body), args[2], args[3])
		if err != nil {
			return err
		}
		return os.WriteFile(args[4], []byte(complete), 0o600)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

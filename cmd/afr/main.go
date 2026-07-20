package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	afr "afr/internal"
)

const version = "0.1.0-dev"

const (
	exitUsage = 64
	exitAFR   = 70
)

func main() {
	os.Exit(realMain(os.Args[1:]))
}

func realMain(args []string) int {
	if len(args) == 0 {
		usage()
		return exitUsage
	}
	switch args[0] {
	case "version":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "AFR_USAGE: version accepts no arguments")
			return exitUsage
		}
		fmt.Println("afr " + version)
		return 0
	case "run":
		return runCommand(args[1:])
	case "list":
		return listCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "AFR_USAGE: unknown command %q\n", args[0])
		usage()
		return exitUsage
	}
}

func listCommand(args []string) int {
	flags := flag.NewFlagSet("afr list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	limit := flags.Int("limit", 20, "maximum sessions")
	asJSON := flags.Bool("json", false, "emit JSON")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() != 0 || *limit < 1 {
		fmt.Fprintln(os.Stderr, "AFR_USAGE: list accepts only --limit N and --json")
		return exitUsage
	}
	root, err := afr.DefaultSessionsRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	sessions, err := afr.ListSessions(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	if len(sessions) > *limit {
		sessions = sessions[:*limit]
	}
	if *asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(sessions); err != nil {
			fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
			return exitAFR
		}
		return 0
	}
	for _, session := range sessions {
		fmt.Printf("%s\t%s\n", session.ID, session.State)
	}
	return 0
}

func runCommand(args []string) int {
	flags := flag.NewFlagSet("afr run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	workspace := flags.String("workspace", "", "workspace directory")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	argv := flags.Args()
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "AFR_USAGE: run requires -- COMMAND [ARG...]")
		return exitUsage
	}
	interrupts := make(chan os.Signal, 2)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	result, err := afr.Run(afr.RunOptions{
		Workspace:  *workspace,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Interrupts: interrupts,
	}, argv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		if result.SessionID != "" {
			fmt.Fprintf(os.Stderr, "AFR session %s incomplete or failed: %s\n", result.SessionID, result.SessionDir)
		}
		return exitAFR
	}
	fmt.Fprintf(os.Stderr, "AFR session %s: %s\n", result.SessionID, result.SessionDir)
	return result.ExitCode
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: afr version | afr run [--workspace PATH] -- COMMAND [ARG...] | afr list [--limit N] [--json]")
}

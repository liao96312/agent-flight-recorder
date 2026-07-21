package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	afr "afr/internal"
)

const version = "0.1.0-dev"

const (
	exitVerify = 2
	exitUsage  = 64
	exitAFR    = 70
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
	case "show":
		return showCommand(args[1:])
	case "clean":
		return cleanCommand(args[1:])
	case "verify":
		return verifyCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "AFR_USAGE: unknown command %q\n", args[0])
		usage()
		return exitUsage
	}
}

func verifyCommand(args []string) int {
	asJSON, selector, err := parseJSONSelector("verify", args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_USAGE: %v\n", err)
		return exitUsage
	}
	sessionsRoot, err := afr.DefaultSessionsRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	sessionRoot, err := afr.ResolveSessionRoot(sessionsRoot, selector)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	result := afr.VerifySession(sessionRoot)
	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
			return exitAFR
		}
	} else if result.Valid {
		fmt.Printf("verified %s: %d events, %d evidence, %d derived\n", result.SessionID, result.Events, result.EvidenceChecked, result.DerivedChecked)
	} else if result.Issue != nil {
		fmt.Fprintf(os.Stderr, "AFR_VERIFY: %s: %s", result.Issue.Code, result.Issue.Message)
		if result.Issue.Seq != 0 {
			fmt.Fprintf(os.Stderr, " (seq %d)", result.Issue.Seq)
		}
		if result.Issue.Path != "" {
			fmt.Fprintf(os.Stderr, " [%s]", result.Issue.Path)
		}
		fmt.Fprintln(os.Stderr)
	}
	if !result.Valid {
		return exitVerify
	}
	return 0
}

func showCommand(args []string) int {
	asJSON, selector, err := parseJSONSelector("show", args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_USAGE: %v\n", err)
		return exitUsage
	}
	sessionsRoot, err := afr.DefaultSessionsRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	sessionRoot, err := afr.ResolveSessionRoot(sessionsRoot, selector)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	result, err := afr.ShowSession(sessionRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
			return exitAFR
		}
		return 0
	}
	if report, err := afr.ReadMarkdownReport(result.MarkdownPath); err == nil {
		_, _ = os.Stdout.Write(report)
		return 0
	}
	fmt.Printf("Session: %s\nState: %s\nStarted: %s\n", result.Session.ID, result.Session.State, result.Session.StartedAt)
	if result.Session.ChildExitCode != nil {
		fmt.Printf("Child exit: %d\n", *result.Session.ChildExitCode)
	}
	if result.Session.TornTailBytes > 0 {
		fmt.Printf("Torn tail: %d bytes\n", result.Session.TornTailBytes)
	}
	return 0
}

func parseJSONSelector(command string, args []string) (bool, string, error) {
	asJSON, selector, selectorSet := false, "latest", false
	for _, argument := range args {
		switch {
		case argument == "--json":
			asJSON = true
		case strings.HasPrefix(argument, "-"):
			return false, "", fmt.Errorf("unknown %s option %q", command, argument)
		case selectorSet:
			return false, "", fmt.Errorf("%s accepts at most one SESSION|latest selector", command)
		default:
			selector, selectorSet = argument, true
		}
	}
	return asJSON, selector, nil
}

func listCommand(args []string) int {
	flags := flag.NewFlagSet("afr list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	limit := flags.Int("limit", 20, "maximum sessions")
	asJSON := flags.Bool("json", false, "emit JSON")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "AFR_USAGE: %v\n", err)
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

func cleanCommand(args []string) int {
	flags := flag.NewFlagSet("afr clean", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	olderText := flags.String("older-than", "", "select sessions older than a duration")
	maxText := flags.String("max-bytes", "", "select oldest sessions until storage is within size")
	yes := flags.Bool("yes", false, "delete without an interactive confirmation")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "AFR_USAGE: %v\n", err)
		return exitUsage
	}
	if flags.NArg() != 0 || (*olderText == "") == (*maxText == "") {
		fmt.Fprintln(os.Stderr, "AFR_USAGE: clean requires exactly one of --older-than DURATION or --max-bytes SIZE")
		return exitUsage
	}
	options := afr.CleanOptions{}
	var err error
	if *olderText != "" {
		options.OlderThan, err = parseCleanDuration(*olderText)
	} else {
		options.MaxBytes, err = parseByteSize(*maxText)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_USAGE: %v\n", err)
		return exitUsage
	}
	root, err := afr.DefaultSessionsRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	plan, err := afr.PlanClean(root, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	fmt.Printf("Clean preview: %d sessions, %d bytes\n", len(plan.Targets), plan.TotalBytes)
	for _, target := range plan.Targets {
		fmt.Printf("%s\t%d\t%s\n", target.ID, target.Bytes, target.Path)
	}
	if len(plan.Targets) == 0 {
		return 0
	}
	if !*yes {
		if !stdinIsTerminal() {
			fmt.Fprintln(os.Stderr, "AFR_USAGE: clean deletion requires an interactive terminal or --yes")
			return exitUsage
		}
		confirmed, confirmErr := confirmClean(os.Stdin, os.Stderr)
		if confirmErr != nil {
			fmt.Fprintf(os.Stderr, "AFR_RUNTIME: read clean confirmation: %v\n", confirmErr)
			return exitAFR
		}
		if !confirmed {
			fmt.Fprintln(os.Stderr, "Clean cancelled; nothing deleted.")
			return 0
		}
	}
	deleted, err := afr.ExecuteClean(plan)
	for _, target := range deleted {
		fmt.Printf("Deleted %s\t%d\t%s\n", target.ID, target.Bytes, target.Path)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "AFR_RUNTIME: %v\n", err)
		return exitAFR
	}
	return 0
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func confirmClean(input io.Reader, output io.Writer) (bool, error) {
	if _, err := fmt.Fprint(output, "Delete the listed sessions? [y/N] "); err != nil {
		return false, err
	}
	answer, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

func parseCleanDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseInt(strings.TrimSuffix(value, "d"), 10, 64)
		if err != nil || days <= 0 || days > int64((1<<63-1)/(24*time.Hour)) {
			return 0, fmt.Errorf("invalid clean duration %q", value)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid clean duration %q", value)
	}
	return duration, nil
}

func parseByteSize(value string) (int64, error) {
	upper := strings.ToUpper(strings.TrimSpace(value))
	multiplier := int64(1)
	for _, suffix := range []struct {
		name       string
		multiplier int64
	}{{"GIB", 1 << 30}, {"MIB", 1 << 20}, {"KIB", 1 << 10}, {"GB", 1_000_000_000}, {"MB", 1_000_000}, {"KB", 1_000}, {"B", 1}} {
		if strings.HasSuffix(upper, suffix.name) {
			upper = strings.TrimSpace(strings.TrimSuffix(upper, suffix.name))
			multiplier = suffix.multiplier
			break
		}
	}
	amount, err := strconv.ParseInt(upper, 10, 64)
	if err != nil || amount <= 0 || amount > (1<<63-1)/multiplier {
		return 0, fmt.Errorf("invalid byte size %q", value)
	}
	return amount * multiplier, nil
}

func runCommand(args []string) int {
	flags := flag.NewFlagSet("afr run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	workspace := flags.String("workspace", "", "workspace directory")
	separator := -1
	for index, argument := range args {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator < 0 {
		fmt.Fprintln(os.Stderr, "AFR_USAGE: run requires -- before COMMAND")
		return exitUsage
	}
	if err := flags.Parse(args[:separator]); err != nil {
		fmt.Fprintf(os.Stderr, "AFR_USAGE: %v\n", err)
		return exitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "AFR_USAGE: run options must appear before --")
		return exitUsage
	}
	argv := args[separator+1:]
	if len(argv) == 0 || argv[0] == "" {
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
	fmt.Fprintln(os.Stderr, "usage: afr version | afr run [--workspace PATH] -- COMMAND [ARG...] | afr list [--limit N] [--json] | afr show [--json] [SESSION|latest] | afr verify [--json] [SESSION|latest] | afr clean (--older-than DURATION | --max-bytes SIZE) [--yes]")
}

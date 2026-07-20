package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("fake-agent", flag.ContinueOnError)
	flags.SetOutput(stderr)
	stdoutText := flags.String("stdout", "", "text written to stdout")
	stderrText := flags.String("stderr", "", "text written to stderr")
	exitCode := flags.Int("exit", 0, "exit code")
	delay := flags.Duration("delay", 0, "delay before actions")
	longLine := flags.Int64("long-line", 0, "number of x bytes written without a newline")
	binary := flags.Bool("binary", false, "write a small binary sample")
	writeFile := flags.String("write-file", "", "file to create")
	writeText := flags.String("write-text", "written", "file contents")
	spawnWriter := flags.String("spawn-writer", "", "spawn a child that writes this file")
	spawnDelay := flags.Duration("spawn-delay", time.Second, "child writer delay")
	selfKill := flags.Bool("self-kill", false, "terminate this process")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	if *longLine < 0 {
		fmt.Fprintln(stderr, "long-line must be non-negative")
		return 64
	}
	time.Sleep(*delay)
	_, _ = io.WriteString(stdout, *stdoutText)
	_, _ = io.WriteString(stderr, *stderrText)
	if *longLine > 0 {
		_, _ = io.CopyN(stdout, repeatedByte('x'), *longLine)
	}
	if *binary {
		_, _ = stdout.Write([]byte{0, 1, 2, 255})
	}
	if *writeFile != "" {
		if err := os.WriteFile(*writeFile, []byte(*writeText), 0o600); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if *spawnWriter != "" {
		command := exec.Command(os.Args[0], "--delay", spawnDelay.String(), "--write-file", *spawnWriter, "--write-text", *writeText)
		if err := command.Start(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if *selfKill {
		process, _ := os.FindProcess(os.Getpid())
		_ = process.Kill()
		time.Sleep(time.Second)
		return 1
	}
	return *exitCode
}

type repeatedByte byte

func (value repeatedByte) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = byte(value)
	}
	return len(buffer), nil
}

// Package cli is the driving adapter: it parses commands and talks to the
// core only through ports.Facade.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"interview-process/internal/core/domain"
	"interview-process/internal/core/ports"
)

const usage = `hncapture: capture Hacker News threads for archiving.

Usage:
  hncapture [capture] [flags]   capture the top stories (default: 5)
  hncapture story <id> [flags]  capture a single story's thread
  hncapture user <id> [flags]   fetch one user profile

Run "hncapture <command> -h" for flags.
`

// errUsage marks errors that should exit with code 2.
var errUsage = errors.New("usage error")

type CLI struct {
	Facade ports.Facade
	Stdout io.Writer
	Stderr io.Writer
}

// Run executes args (without the program name) and returns an exit code:
// 0 success, 1 runtime failure, 2 usage error.
func (c CLI) Run(ctx context.Context, args []string) int {
	cmd := "capture"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}

	var err error
	switch cmd {
	case "capture":
		err = c.capture(ctx, args)
	case "story":
		err = c.story(ctx, args)
	case "user":
		err = c.user(ctx, args)
	case "help":
		fmt.Fprint(c.Stdout, usage)
		return 0
	default:
		fmt.Fprint(c.Stderr, usage)
		err = fmt.Errorf("%w: unknown command %q", errUsage, cmd)
	}

	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errUsage):
		fmt.Fprintln(c.Stderr, "error:", err)
		return 2
	default:
		fmt.Fprintln(c.Stderr, "error:", err)
		return 1
	}
}

// captureFlags are shared by capture and story.
type captureFlags struct {
	out          string
	concurrency  int
	timeout      time.Duration
	format       string
	fullProfiles bool
}

func (c CLI) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	return fs
}

func (f *captureFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&f.out, "out", "output", `directory for the JSON archive ("" to skip saving)`)
	fs.IntVar(&f.concurrency, "concurrency", 16, "max in-flight API requests")
	fs.DurationVar(&f.timeout, "timeout", 5*time.Minute, "overall timeout")
	fs.StringVar(&f.format, "format", "tree", "stdout format: tree, json or none")
	fs.BoolVar(&f.fullProfiles, "full-profiles", false, "keep each author's full submission id list (large)")
}

func (f *captureFlags) validate() error {
	switch f.format {
	case "tree", "json", "none":
	default:
		return fmt.Errorf("%w: --format must be tree, json or none", errUsage)
	}
	if f.concurrency < 1 {
		return fmt.Errorf("%w: --concurrency must be at least 1", errUsage)
	}
	return nil
}

func (c CLI) capture(ctx context.Context, args []string) error {
	fs := c.newFlagSet("capture")
	var f captureFlags
	f.register(fs)
	top := fs.Int("top", 5, "number of top stories to capture")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	if err := f.validate(); err != nil {
		return err
	}
	if *top < 1 {
		return fmt.Errorf("%w: --top must be at least 1", errUsage)
	}
	return c.runCapture(ctx, f, func(ctx context.Context, o ports.CaptureOptions) (domain.Capture, error) {
		return c.Facade.CaptureTop(ctx, *top, o)
	})
}

func (c CLI) story(ctx context.Context, args []string) error {
	fs := c.newFlagSet("story")
	var f captureFlags
	f.register(fs)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return fmt.Errorf("%w: story needs exactly one item id", errUsage)
	}
	id, err := strconv.Atoi(pos[0])
	if err != nil || id < 1 {
		return fmt.Errorf("%w: invalid item id %q", errUsage, pos[0])
	}
	if err := f.validate(); err != nil {
		return err
	}
	return c.runCapture(ctx, f, func(ctx context.Context, o ports.CaptureOptions) (domain.Capture, error) {
		return c.Facade.CaptureStory(ctx, id, o)
	})
}

func (c CLI) runCapture(ctx context.Context, f captureFlags, do func(context.Context, ports.CaptureOptions) (domain.Capture, error)) error {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	start := time.Now()
	capture, err := do(ctx, ports.CaptureOptions{Concurrency: f.concurrency, FullProfiles: f.fullProfiles})
	if err != nil {
		return err
	}
	elapsed := time.Since(start)

	switch f.format {
	case "tree":
		renderCapture(c.Stdout, capture)
	case "json":
		if err := writeJSON(c.Stdout, capture); err != nil {
			return err
		}
	}

	renderSummary(c.Stderr, capture, elapsed)
	if f.out != "" {
		path, err := c.Facade.Save(ctx, capture, f.out)
		if err != nil {
			return fmt.Errorf("save archive: %w", err)
		}
		fmt.Fprintln(c.Stderr, "archive written to", path)
	}
	return nil
}

func (c CLI) user(ctx context.Context, args []string) error {
	fs := c.newFlagSet("user")
	format := fs.String("format", "tree", "stdout format: tree or json")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return fmt.Errorf("%w: user needs exactly one username", errUsage)
	}
	if *format != "tree" && *format != "json" {
		return fmt.Errorf("%w: --format must be tree or json", errUsage)
	}
	u, err := c.Facade.GetUser(ctx, pos[0])
	if err != nil {
		return err
	}
	if *format == "json" {
		return writeJSON(c.Stdout, u)
	}
	renderUser(c.Stdout, u)
	return nil
}

// parse parses flags allowing them before or after positional arguments
// (e.g. "story 8863 --out dir"), returning the positionals.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, fmt.Errorf("%w: %v", errUsage, err)
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos, args = append(pos, args[0]), args[1:]
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

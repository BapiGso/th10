// Command wasmperf measures CPU-side WASM streaming and headless logic.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"th10/sim"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "wasmperf:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("wasmperf", flag.ContinueOnError)
	kind := flags.String("kind", "logic", "measurement: logic or audio")
	blocks := flags.Int("blocks", 32, "sequential reads per track/buffer size")
	replayPath := flags.String("replay", "", "input replay for logic measurement")
	chapter := flags.Int("chapter", 0, "replay chapter index")
	frames := flags.Int("frames", 0, "logic frame limit (0 = all)")
	warmup := flags.Int("warmup", 0, "untimed fresh logic runs before measurement (0 = cold)")
	output := flags.String("out", "-", "JSON output path (- = stdout)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *blocks <= 0 || *frames < 0 || *warmup < 0 {
		return fmt.Errorf("invalid kind/blocks/arguments")
	}
	var measurements []audioMeasurement
	var logic *sim.Profile
	var err error
	switch *kind {
	case "audio":
		measurements, err = measureAudio(*blocks)
	case "logic":
		if *replayPath == "" {
			return fmt.Errorf("logic measurement requires -replay")
		}
		logic, err = measureLogic(*replayPath, *chapter, *frames, *warmup)
	default:
		return fmt.Errorf("unknown measurement %q", *kind)
	}
	if err != nil {
		return err
	}
	var out io.Writer = os.Stdout
	if *output != "-" {
		file, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		out = file
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(struct {
		Kind, GoVersion, GOOS, GOARCH string
		Audio                         []audioMeasurement
		Logic                         *sim.Profile
		Warmup                        int
	}{*kind, runtime.Version(), runtime.GOOS, runtime.GOARCH, measurements, logic, *warmup})
}

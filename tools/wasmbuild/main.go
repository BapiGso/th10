// Command wasmbuild builds the embedded game and optionally previews it over HTTP.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

//go:embed web/index.html web/game.html web/app.mjs web/player.mjs web/helpers.mjs
var webFiles embed.FS

var webNames = []string{"index.html", "game.html", "app.mjs", "player.mjs", "helpers.mjs"}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "wasmbuild:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("wasmbuild", flag.ContinueOnError)
	out := flags.String("out", "dist", "output directory (relative to the module root)")
	serve := flags.String("serve", "", "preview address, e.g. :8080 (bare ports bind loopback only)")
	skip := flags.Bool("skip-build", false, "serve an existing release without rebuilding")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*skip && *serve == "") {
		return errors.New("unexpected arguments, or -skip-build without -serve")
	}
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	dir := *out
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(root, dir)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}
	if !*skip {
		goroot, version, err := goToolchain(ctx, root)
		if err != nil {
			return err
		}
		m, err := buildRelease(ctx, releaseOptions{Root: root, Out: dir, GoRoot: goroot, GoVersion: version}, buildWASM)
		if err != nil {
			return err
		}
		fmt.Printf("%s\nth10.wasm: %d bytes (%.2f MiB)\nth10.wasm.gz: %d bytes (%.2f MiB)\nGo: %s\n",
			dir, m.WASMBytes, float64(m.WASMBytes)/(1<<20), m.GzipBytes, float64(m.GzipBytes)/(1<<20), m.GoVersion)
	}
	if *serve == "" {
		return nil
	}
	handler, err := previewHandler(dir)
	if err != nil {
		return err
	}
	address := *serve
	if strings.HasPrefix(address, ":") {
		address = "127.0.0.1" + address
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: time.Minute}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	fmt.Printf("Preview: http://%s/ (Ctrl+C to stop; keep this origin/port for saved data)\n", listener.Addr())
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("run within the th10 module")
		}
		dir = parent
	}
}

func goToolchain(ctx context.Context, root string) (string, string, error) {
	command := exec.CommandContext(ctx, "go", "env", "GOROOT", "GOVERSION")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return "", "", fmt.Errorf("go env: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		return "", "", errors.New("unexpected go env GOROOT GOVERSION output")
	}
	return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1]), nil
}

func buildWASM(ctx context.Context, root, output string) error {
	command := exec.CommandContext(ctx, "go", "build", "-trimpath", "-ldflags=-s -w", "-o", output, "./cmd")
	command.Dir = root
	// Preserve the caller's sandbox/cache settings, but do not inherit a host target.
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(name, "GOOS") && !strings.EqualFold(name, "GOARCH") {
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, "GOOS=js", "GOARCH=wasm")
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("WASM build: %w", err)
	}
	return nil
}

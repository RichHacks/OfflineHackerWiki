package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"hackerwiki/internal/config"
	"hackerwiki/internal/server"
)

func main() {
	cfg := config.Parse(resolveContentBase())

	srv := server.New(cfg)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("HackerWiki listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}

// resolveContentBase finds the content/ directory relative to the binary or CWD.
func resolveContentBase() string {
	// 1. Explicit env override
	if p := os.Getenv("HACKERWIKI_CONTENT"); p != "" {
		return p
	}
	// 2. content/ relative to CWD
	cwd, _ := os.Getwd()
	if p := filepath.Join(cwd, "content"); dirExists(p) {
		return p
	}
	// 3. content/ relative to binary location
	exe, err := os.Executable()
	if err == nil {
		if p := filepath.Join(filepath.Dir(exe), "content"); dirExists(p) {
			return p
		}
	}
	// 4. Fallback: source file directory (useful during `go run`)
	_, file, _, ok := runtime.Caller(0)
	if ok {
		if p := filepath.Join(filepath.Dir(file), "content"); dirExists(p) {
			return p
		}
	}
	return ""
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: hackerwiki [options]\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Content is auto-discovered from ./content/ next to the binary or CWD.
Override individual paths with flags or set HACKERWIKI_CONTENT=/path/to/content/.
Update all content: git submodule update --remote
`)
	}
}

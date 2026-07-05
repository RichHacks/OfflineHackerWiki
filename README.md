# HackerWiki

A Go server that mirrors a handful of security reference sites/tools locally, each mounted under its own path (HackTricks, PayloadsAllTheThings, Revshells, PortSwigger's XSS cheat sheet, Certipy's wiki, CyberChef).

## First-time setup

This repo isn't pushed anywhere yet — there's no remote to clone from. Once it is, the process for a new machine will be:

```
git clone --recurse-submodules <this-repo-url>
cd HackerWiki
go run .
```


Open http://localhost:8888. Custom port: `go run . -port 9000`.

If you ever end up with an empty `content/<name>` directory (e.g. after a plain `git clone` without `--recurse-submodules`):

```
git submodule update --init --recursive
```

## Building a binary

```
scripts/build.sh          # cross-compiles to dist/: linux/darwin/windows, amd64/arm64
```


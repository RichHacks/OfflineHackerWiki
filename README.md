# HackerWiki

A Go server that mirrors a handful of security reference sites/tools locally, each mounted under its own path (HackTricks, PayloadsAllTheThings, Revshells, PortSwigger's XSS cheat sheet, Certipy's wiki, CyberChef). Provides a landing page so you can navigate between them all, and use each in offline environments. Also does not rely on running multiple containers, its just an executable.

![HackerWiki landing page](./images/hackerwiki.png)

## First-time setup

```
git clone --recurse-submodules git@github.com:RichHacks/OfflineHackerWiki.git
cd OfflineHackerWiki
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


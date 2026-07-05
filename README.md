# HackerWiki

A Go server that mirrors a handful of security reference sites/tools locally, each mounted under its own path (HackTricks, PayloadsAllTheThings, Revshells, PortSwigger's XSS cheat sheet, Certipy's wiki, CyberChef). Provides a landing page so you can navigate between them all, and use each in offline environments. Also does not rely on running multiple containers, its just an executable.

![HackerWiki landing page](./images/hackerwiki.png)

## First-time setup

```
git clone --recurse-submodules git@github.com:RichHacks/OfflineHackerWiki.git
cd OfflineHackerWiki
scripts/build.sh
```

That cross-compiles binaries for Linux, macOS, and Windows (amd64 + arm64) into `dist/`. Run the one for your machine, e.g.:

```
./dist/hackerwiki-darwin-arm64
```

Open http://localhost:8888. Custom port: `./dist/hackerwiki-darwin-arm64 -port 9000`.

If you ever end up with an empty `content/<name>` directory (e.g. after a plain `git clone` without `--recurse-submodules`):

```
git submodule update --init --recursive
```


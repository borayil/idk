# idk

Type `idk` when you forgot the command. Get a short numbered list, pick one.

```
$ idk
  1) go build ./...
  2) git status
  3) npm install
  4) go test ./...
  5) git push

run: 1-9   edit: e then 1-9   cancel: 0 or ctrl+c
```

- digit → runs it
- `e` then a digit (e.g. `e2`) → drops it into your prompt line, editable, doesn't run
- `0` or ctrl+c → cancel

No AI, no network, by default. Instant.

## Install

```
git clone <this-repo>
cd idk
go build -o idk-bin .
mv idk-bin ~/.local/bin/
```

Add one line to your shell rc:

```sh
eval "$(idk-bin init zsh)"    # ~/.zshrc
eval "$(idk-bin init bash)"   # ~/.bashrc
```

New shell (or `source` it) and `idk` works.

## How it picks

No ML, just arithmetic. Every past command gets a score from: recency (decays
over ~2 weeks), how often you've run it, whether you've run it in *this exact
directory* before (weighted heavier, decays faster), and whether it matches
what's going on right now (touched a `.go` file → `go build`/`go test` get a
boost, same idea for npm/pip/terraform/gcloud). Top N win. Ties are broken
deterministically so the list doesn't reshuffle for no reason between runs.

## Optional AI

Off by default. Turn it on and it tosses in 1-2 extra picks from an LLM,
always labeled `(ai, unverified)` since — unlike the rest of the list —
those are generated text, not commands you've actually run:

```
export ANTHROPIC_API_KEY=...   # or OPENAI_API_KEY
idk-bin enable ai
idk-bin disable ai
```

The key lives in your env only, never touches disk. Capped at 1.5s and fails
quietly back to normal suggestions if it doesn't respond in time.

## Config

```
idk -n 7        # show 7 this run
IDK_N=3 idk     # or via env var
```

3-9, default 5.

## Limitations

- bash's edit mode is a fallback (`history -s`, press ↑) — zsh gets a real
  inline prefill via `print -z`, bash doesn't have an equivalent.
- zsh and bash only for now.

## License

MIT

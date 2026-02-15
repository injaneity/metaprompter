# Homebrew Install

## Option 1: Install from this local repo

```bash
brew install --HEAD ./Formula/metaprompter.rb
```

## Option 2: Install from raw formula URL

```bash
brew install --HEAD https://raw.githubusercontent.com/injaneity/metaprompter/main/Formula/metaprompter.rb
```

## Option 3: Install via your own tap (recommended for team use)

1. Create a tap repo on GitHub named `homebrew-metaprompter`.
2. Copy `Formula/metaprompter.rb` into that tap repo under `Formula/`.
3. Install from tap:

```bash
brew tap injaneity/metaprompter
brew install --HEAD injaneity/metaprompter/metaprompter
```

## Upgrade

```bash
brew upgrade --fetch-HEAD metaprompter
```

## Verify

```bash
metaprompter --help
```

## Notes

- This formula is currently `head`-based, so it builds from source using the latest `main`.
- `go` is installed automatically as a build dependency by Homebrew.

## Stable release formula (`url + sha256`)

After creating a GitHub release tag (example: `v0.1.0`), run:

```bash
./scripts/update_formula_release.sh v0.1.0
```

This updates `Formula/metaprompter.rb` to:
- `url "https://github.com/<owner>/<repo>/archive/refs/tags/<tag>.tar.gz"`
- `sha256 "<computed hash>"`
- keeps `head` for `--HEAD` installs

Then commit the formula update and push.

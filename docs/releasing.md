# Releasing

## 1) Tag and push a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

Create a GitHub Release for that tag.

## 2) Update Homebrew stable formula

```bash
./scripts/update_formula_release.sh v0.1.0
```

This computes the tarball hash and writes `Formula/metaprompter.rb` with stable `url` + `sha256`.

## 3) Commit formula update

```bash
git add Formula/metaprompter.rb
git commit -m "brew: update stable formula for v0.1.0"
git push
```

## 4) Install test

```bash
brew install https://raw.githubusercontent.com/injaneity/metaprompter/main/Formula/metaprompter.rb
metaprompter --help
```

## Notes

- Use `--HEAD` to install latest main branch build.
- Stable installs require a published GitHub release tag.

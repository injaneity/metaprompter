#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "Usage: $0 <tag> [owner/repo]" >&2
  echo "Example: $0 v0.1.0 injaneity/metaprompter" >&2
  exit 1
fi

tag="$1"
repo="${2:-}"

if [[ -z "$repo" ]]; then
  remote_url="$(git remote get-url origin)"
  case "$remote_url" in
    git@github.com:*)
      repo="${remote_url#git@github.com:}"
      ;;
    https://github.com/*)
      repo="${remote_url#https://github.com/}"
      ;;
    http://github.com/*)
      repo="${remote_url#http://github.com/}"
      ;;
    *)
      echo "Could not infer GitHub repo from origin: $remote_url" >&2
      echo "Pass it explicitly as owner/repo." >&2
      exit 1
      ;;
  esac
  repo="${repo%.git}"
fi

if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-].*)?$ ]]; then
  echo "Tag should look like v0.1.0 (or v0.1.0-rc1). Got: $tag" >&2
  exit 1
fi

archive_url="https://github.com/${repo}/archive/refs/tags/${tag}.tar.gz"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
archive="$tmpdir/${tag}.tar.gz"

curl -fsSL "$archive_url" -o "$archive"
sha256="$(shasum -a 256 "$archive" | awk '{print $1}')"

cat > Formula/metaprompter.rb <<FORMULA
class Metaprompter < Formula
  desc "Specs-first metaprompting CLI for agentic coding workflows"
  homepage "https://github.com/${repo}"
  url "${archive_url}"
  sha256 "${sha256}"
  head "https://github.com/${repo}.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/metaprompter"
  end

  test do
    assert_match "metaprompter commands", shell_output("#{bin}/metaprompter --help")
  end
end
FORMULA

echo "Updated Formula/metaprompter.rb for ${repo} ${tag}"
echo "url: ${archive_url}"
echo "sha256: ${sha256}"

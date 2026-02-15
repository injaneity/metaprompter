class Metaprompter < Formula
  desc "Specs-first metaprompting CLI for agentic coding workflows"
  homepage "https://github.com/injaneity/metaprompter"
  head "https://github.com/injaneity/metaprompter.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/metaprompter"
  end

  test do
    assert_match "metaprompter commands", shell_output("#{bin}/metaprompter --help")
  end
end

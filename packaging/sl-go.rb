# Homebrew formula.  To use it from a tap:
#   brew install --build-from-source ./packaging/sl-go.rb
# and for a tap, copy it to Formula/sl-go.rb with the url/sha256 of the release
# tarball (packaging/make-packages.sh prints both).
#
# homebrew-core carries "sl" (the steam locomotive) in the main tap; the same
# binary path is installed, hence conflicts_with.  Go is a build dependency
# only: the result is one static Mach-O executable.
class SlGo < Formula
  desc "More useful ls: group files by purpose and point out interesting ones"
  homepage "https://web.archive.org/web/2016/http://practicalthought.com/sl/"
  url "https://example.invalid/sl-go/sl-go-@VERSION@.tar.gz"
  sha256 "@SHA256@"
  license "GPL-3.0-or-later"
  version "@VERSION@"

  depends_on "go" => :build
  conflicts_with "sl", because: "both install a binary named sl"

  def install
    ENV["CGO_ENABLED"] = "0"
    system "go", "build", "-trimpath", "-ldflags", "-s -w", "-o", "sl", "./src"
    bin.install "sl"
    man1.install "docs/sl.1"
    bash_completion.install "completions/sl.bash" => "sl"
    zsh_completion.install "completions/_sl"
    fish_completion.install "completions/sl.fish"
    doc.install "README.md", "INSTALL.md", "LICENSE"
  end

  test do
    assert_match "@VERSION@", shell_output("#{bin}/sl --version")
    # --color=never must print nothing escape-like to a pipe
    assert_no_match(/\e\[/, shell_output("#{bin}/sl --color=never -1 #{testpath}"))
    # a user rule has to take effect
    (testpath/"note.md").write "x\n"
    assert_match "doc", shell_output("#{bin}/sl --no-color -1 --rule '\\.md$ -> doc' #{testpath}")
  end
end

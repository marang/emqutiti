class Emqutiti < Formula
  desc "Terminal MQTT client"
  homepage "https://github.com/marang/emqutiti"
  url "https://github.com/marang/emqutiti/archive/refs/tags/v0.4.1.tar.gz"
  sha256 "ce8ab0d28762d6ed6d7284bd6d3a774225339a2696bf277a2f2d74bc65ed62bd"
  license "MIT"
  head "https://github.com/marang/emqutiti.git"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args, "./cmd/emqutiti"
  end

  test do
    output = shell_output("#{bin}/emqutiti -h", 2)
    assert_match "Usage", output
  end
end

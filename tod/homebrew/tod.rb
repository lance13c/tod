cask "tod" do
  version "0.0.10"
  sha256 "1f17b03370af05a484411ecc1e1deb345c711b66a285422bd07eecd630d27a10"

  url "https://github.com/lance13c/tod/releases/download/v#{version}/tod-#{version}.dmg"
  name "Tod"
  desc "Agentic TUI manual tester - A text-adventure interface for E2E testing"
  homepage "https://github.com/lance13c/tod"

  livecheck do
    url :url
    strategy :github_latest
  end

  app "Tod.app"

  # Add tod binary to PATH by creating a symlink
  binary "#{appdir}/Tod.app/Contents/MacOS/tod"

  zap trash: [
    "~/Library/Application Support/tod",
    "~/Library/Caches/tod",
    "~/Library/Preferences/com.ciciliostudio.tod.plist",
  ]
end
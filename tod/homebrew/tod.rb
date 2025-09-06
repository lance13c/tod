cask "tod" do
  version "0.0.1"
  sha256 :no_check # Will be updated when v0.0.1 DMG is built

  url "https://github.com/ciciliostudio/tod/releases/download/v#{version}/tod-#{version}.dmg"
  name "Tod"
  desc "Agentic TUI manual tester - A text-adventure interface for E2E testing"
  homepage "https://github.com/ciciliostudio/tod"

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
cask "tod" do
  version "0.0.7"
  sha256 "1adc16bbfacad8958b2b448528f259a1e5e7ecbdc9077108163e55b8c59b747e"

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
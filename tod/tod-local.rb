cask "tod-local" do
  version "0.0.2"
  sha256 "69be67fce303495cd4cb4f1db47e3db9ae2ba0f933579bf0a7d8083f0b4aac66"

  url "file:///Users/dominic.cicilio/Documents/repos/test-god/tod/dist/tod-0.0.2.dmg"
  name "Tod"
  desc "Agentic TUI manual tester - A text-adventure interface for E2E testing"
  homepage "https://tod.dev"

  app "Tod.app"

  # Add tod binary to PATH by creating a symlink
  binary "#{appdir}/Tod.app/Contents/MacOS/tod"

  zap trash: [
    "~/Library/Application Support/tod",
    "~/Library/Caches/tod",
    "~/Library/Preferences/com.ciciliostudio.tod.plist",
  ]
end
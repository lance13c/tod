cask "tod" do
  version "0.0.2"
  sha256 "cabd1b6afc631189cf61a1aa008fdd6a1a06d4b04b7d0814e728666d4512c4b6"

  url "https://github.com/lance13c/tod/releases/download/v#{version}/tod-#{version}.dmg",
      verified: "github.com/lance13c/tod/"
  name "Tod"
  desc "Agentic TUI manual tester - A text-adventure interface for E2E testing"
  homepage "https://tod.dev/"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on macos: ">= :catalina"

  app "Tod.app"
  binary "#{appdir}/Tod.app/Contents/MacOS/tod"

  zap trash: [
    "~/Library/Application Support/tod",
    "~/Library/Caches/tod",
    "~/Library/Preferences/com.ciciliostudio.tod.plist",
  ]
end

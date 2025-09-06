cask "tod" do
  version "0.0.4"
  sha256 "1a4dc436130fa8ad18d397e6d0a72fab25d0b63ed23320f64a1eb20a30b9f169"

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

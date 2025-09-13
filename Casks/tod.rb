cask "tod" do
  version "0.0.10"
  sha256 "bd63a222eb881bfca3784ee98744ee918b23bf10dd4f3be386fdcfea42e9883a"

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
    "~/Library/Preferences/com.tod.app.plist",
  ]
end

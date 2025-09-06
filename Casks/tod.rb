cask "tod" do
  version "0.0.6"
  sha256 "96dbcda8501c129350da3fb8c7aac6e40935c415305d49c63cd5f3691ed47093"

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

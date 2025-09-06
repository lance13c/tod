cask "tod" do
  version "0.0.3"
  sha256 "4b885d656878337d4ac398395ed1256f01edcd448f8e5f2b0c5eb510f339746f"

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

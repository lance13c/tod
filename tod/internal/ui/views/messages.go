package views

// Message types used by views
type ChromeLaunchedMsg struct{}
type ChromeErrorMsg struct {
	Error error
}
type NavigationCompleteMsg struct {
	URL     string
	Success bool
	Error   error
}
type ReturnToMenuMsg struct{}
type RestartConfigMsg struct{}
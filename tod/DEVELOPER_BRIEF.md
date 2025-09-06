# Tod Developer Brief

## Quick Start
The setup script and `make hotdev` allows you to call `toddev` from anywhere on your system.

## Current Status & Warnings

### Scanning Feature
Scanning is in progress, but **be careful** - scanning can cost a lot. I've put up some guard rails but it isn't perfect. We might switch to a smarter approach: using AI to understand where to look for routes and pages might be more efficient. This isn't pressing though.

## Main Focus
The main thing is to have a good UI that allows the user to navigate around a webapp. The initial work is done.

### Navigation Design
The design of navigation uses a built-in navigation library for Go, then tracks these steps and rebuilds them in the user's e2e test framework. Note: We don't use the user's e2e framework to navigate (which is what I thought we would do), but instead use an internal framework which is a more reliable way to navigate.

## Development Priorities

### 1. Navigation Polish
After navigation feels good - good history logs, and the feel is good - move onto auth.

### 2. Authentication
There's a lot of stuff already made for auth. It just needs to be brought together. There's an email checker (I haven't tried the implementation yet), so probably a lot of work here, but a ton of stuff is setup. 

**Focus on:**
- Magic Links
- Email + Password

### 3. E2E Test Generation
After those two auths work, move onto the e2e test generation logic. After we get to a point in the session of manually navigating, we may want to make that an e2e test. Try to get that e2e test generation working where it generates the test and then we test and fix it agentically.

Might be easiest to just make this for non-interactive mode first.

## Non-Interactive Mode
Non-interactive mode is when it returns one output, performs one action. This allows Claude Code to use all the commands.

All commands should have a non-interactive mode (might need better term). For the e2e test generation, it might be easiest to create the test and tell in the output how to run it, then Claude Code can take over from there. At least to speed this up, then we can implement the agentic fix for when we are in interactive mode.

## Marketing
For marketing stuff, it would be cool to have one or some videos on TikTok. Use the X channel and have fun.

## Distribution & Deployment

### Homebrew
I looked at how to get it on brew and it requires:
- 30 forks
- 30 watches  
- 75 stars

So I instead opted for tap which just requires one more command for it to work.

### Mac Developer Signature
When deploying, you will need a Mac developer signature as it is technically a Mac app. This requires an Apple developer account. I have been using mine, but if you want to deploy a new version to the `lance13c/tod` channel, you will need to sign it otherwise others won't be able to install it.

**Note:** I gave Michael access to lance13c/tod.

### Repositories
- **lance13c/tod** - The public repo (ONLY FOR RELEASES)
- **lance13c/tod-dev** - The private repo we have been working from (I have just been changing the names around). Keep code in tod-dev please.

### Platform Support
I don't know if I made this capable of being downloaded on Windows or Linux yet, I don't remember.

---

I'll be around 3pm, mind is fried.
# API Call Logging for Tod

## Current State

The test generation feature (pressing 'g' after HTML capture) is now fully implemented with comprehensive logging. However, the OpenAI client currently delegates to a mock implementation that returns sample test code rather than making real API calls.

## Logging Implementation

API calls are logged to `.tod/api_calls.log` with detailed information including:
- Timestamp with microseconds
- Component making the call ([TEST_GEN], [ACTION_DISCOVERY], [OPENAI_CLIENT], [MOCK_CLIENT])
- Request details (prompt length, first 500 chars of prompt)
- Response details (success/error, response length)

## How to Monitor API Calls

1. **Start Log Monitoring:**
   ```bash
   ./test_logging.sh
   ```
   This will clear the log file and tail it in real-time.

2. **Run Tod and Test:**
   - Launch Tod: `./tod`
   - Navigate to "Chrome Test Discovery"
   - Let it capture HTML and analyze actions
   - Press 'g' to generate tests
   - Check the log output in the monitoring terminal

## Log Locations

- API call logs: `.tod/api_calls.log`
- Database (including LLM interactions): `.tod/captures.db`

## What You'll See in Logs

When pressing 'g' to generate tests, you should see:
1. `[TEST_GEN]` - Starting test generation
2. `[TEST_GEN]` - Provider and model information
3. `[ACTION_DISCOVERY]` - Test generation request with prompt
4. `[OPENAI_CLIENT]` - Delegation to mock (currently)
5. `[MOCK_CLIENT]` - Mock response with sample test code

## Known Issues

1. **OpenAI Client Uses Mock**: The OpenAI client (`internal/llm/openai.go`) currently delegates all calls to the mock client instead of making real API calls to OpenAI.

2. **Mock Returns Static Tests**: The mock client returns static sample test code regardless of the discovered actions.

## Next Steps

To enable real OpenAI API calls:
1. Implement proper OpenAI API client in `internal/llm/openai.go`
2. Use the OpenAI Go SDK or make direct HTTP calls to the OpenAI API
3. Handle API key validation and error responses
4. Implement proper token counting and cost calculation

## Test Code Generation Flow

1. User presses 'g' in ChromeDebuggerView
2. `generateTestsCmd()` is called
3. Discovered actions are converted to test generation format
4. `GenerateTestSuggestions()` builds a prompt with the actions
5. LLM client's `AnalyzeCode()` is called with the prompt
6. Response is saved to file and displayed in viewport
7. All interactions are logged to both file and database
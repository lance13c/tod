# OpenAI API Setup for Tod

## Configuration

Tod now uses real OpenAI API calls for test generation. To enable this feature, you need to configure your OpenAI API key.

## Setting Up Your Configuration

1. **Create or edit `.tod/config.yaml`:**

```yaml
ai:
  provider: openai
  api_key: YOUR_OPENAI_API_KEY_HERE
  model: gpt-4o-mini  # Cost-efficient model, good for test generation
  settings:
    temperature: 0.7
    max_tokens: 2000

testing:
  framework: playwright  # or cypress, puppeteer, etc.
  language: javascript
  test_dir: tests

environments:
  local:
    name: Local Development
    base_url: http://localhost:3000

current_env: local
```

2. **Alternative Models:**
   - `gpt-4o-mini` - Most cost-efficient ($0.15/1M input, $0.60/1M output)
   - `gpt-4o` - More capable but costlier ($3.00/1M input, $10.00/1M output)
   - `gpt-4-turbo` - Most capable ($10.00/1M input, $30.00/1M output)

## Getting an OpenAI API Key

1. Go to https://platform.openai.com/
2. Sign up or log in
3. Navigate to API Keys section
4. Create a new API key
5. Copy the key and add it to your config

## Cost Estimates

For typical test generation:
- **Input**: ~2000 tokens (prompt with actions)
- **Output**: ~1000 tokens (generated test code)

With `gpt-4o-mini`:
- Cost per test generation: ~$0.001 (less than 1/10 of a cent)

## Testing the Integration

1. Run Tod: `./tod`
2. Select "Chrome Test Discovery"
3. Let it capture and analyze the page
4. Press 'g' to generate tests
5. Monitor `.tod/api_calls.log` for API activity

## Troubleshooting

### No response when pressing 'g'
- Check `.tod/api_calls.log` for errors
- Verify your API key is valid
- Ensure you have credits in your OpenAI account

### API Errors
- **401 Unauthorized**: Invalid API key
- **429 Rate Limit**: Too many requests, wait a moment
- **402 Payment Required**: Add credits to your OpenAI account

### Monitoring Usage
- Check token usage in `.tod/api_calls.log`
- View costs at https://platform.openai.com/usage

## Security Note

**Never commit your API key to version control!**

Consider using environment variables:
```bash
export OPENAI_API_KEY=sk-...
```

Then in config:
```yaml
ai:
  provider: openai
  api_key: ${OPENAI_API_KEY}  # Tod will read from env
  model: gpt-4o-mini
```
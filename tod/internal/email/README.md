# Tod Email Checker Module

The Email Checker module enables Tod to automatically extract authentication data from Gmail emails, supporting magic links, verification codes, and 2FA tokens.

## Quick Start

### 1. One-Time Setup
Configure Gmail access (only needed once per project):
```bash
tod auth setup-email
```

This will:
- Open your browser for Google OAuth
- Request read-only Gmail access  
- Store encrypted credentials in `.tod/credentials/`
- Test the connection

### 2. Use in Test Users
Simply add an email address to any test user:
```yaml
users:
  my_user:
    email: your@gmail.com  # That's it!
    auth_type: magic_link
```

Tod will automatically:
- Wait for authentication emails after auth actions
- Extract verification codes, magic links, or 2FA tokens
- Use them to complete authentication

## Supported Authentication Types

### Magic Links
```yaml
auth_type: magic_link
auth_config:
  email_check_enabled: true  # Default: true
  email_timeout: 30         # Default: 30 seconds
```

### Verification Codes  
```yaml
auth_type: email_verification  # or 2fa, sms
auth_config:
  email_check_enabled: true
  email_timeout: 45
```

## How It Works

1. **Action Trigger**: User action requires email auth (magic link, verification code)
2. **Email Polling**: Tod monitors Gmail for new emails (last 60 seconds)
3. **LLM Analysis**: Uses LLM to intelligently extract auth data from any email format
4. **Auto-Use**: Automatically uses extracted data to complete authentication
5. **Fallback**: Includes regex patterns for common formats when LLM isn't available

## Commands

### Setup & Management
```bash
# Initial setup
tod auth setup-email

# Check status
tod auth status

# Reset configuration  
tod auth reset-email

# Force reconfigure
tod auth setup-email --reset
```

### Create Email-Enabled Users
```bash
# From magic link template (includes email checking)
tod users create --template magic_link_user

# From file with email users
tod users batch --file examples/email-auth-users.yaml
```

## Configuration Options

### Test User Config
```yaml
auth_config:
  email_check_enabled: true|false  # Enable/disable email checking
  email_timeout: 30               # Timeout in seconds (default: 30)
```

### Email Credentials (Auto-Generated)
Stored in `.tod/credentials/email.json`:
```json
{
  "email": "your@gmail.com",
  "refresh_token": "encrypted_token", 
  "client_id": "oauth_client_id",
  "client_secret": "oauth_client_secret"
}
```

## Smart Features

### 🧠 **LLM-Powered Extraction**
- Works with any email template or format
- Understands context (magic link vs verification code)
- No hardcoded patterns required

### ⚡ **Auto-Detection**
- Knows when to check email based on auth type
- Starts checking immediately after auth action
- Stops when data found or timeout reached

### 🔒 **Security**
- Read-only Gmail access
- Encrypted credential storage
- No email content logged or stored
- OAuth refresh tokens never exposed

### 🎯 **Smart Timing**
- Waits for emails to arrive (configurable timeout)
- Polls every 2 seconds for new messages
- Filters to recent emails only (last 60 seconds)

## Usage Examples

### Complete Magic Link Flow
```bash
# 1. Setup email (one-time)
tod auth setup-email

# 2. Create magic link user
tod users create --template magic_link_user
# Enter your Gmail address when prompted

# 3. Start testing
tod
# Tod will automatically handle magic link emails!
```

### Batch User Creation
```bash
# Use the example file as a template
cp examples/email-auth-users.yaml my-users.yaml
# Edit my-users.yaml with your Gmail address
tod users batch --file my-users.yaml
```

### Manual Integration
```go
// In Go code
authFlow, _ := users.NewAuthFlowManager(projectDir, llmClient)
result := authFlow.AuthenticateWithEmailSupport(user)
if result.Success {
    fmt.Printf("Magic link: %s\n", result.RedirectURL)
}
```

## Troubleshooting

### Email Not Configured
```
📧 Email checker is not configured
💡 Run 'tod auth setup-email' to get started
```
**Solution**: Run the setup command to configure Gmail access.

### Connection Issues
```
❌ Email configuration error: failed to test Gmail connection
```
**Solution**: Internet connection issue or expired tokens. Run `tod auth setup-email --reset`.

### No Emails Found
```
📧 Waiting for magic link email...
❌ Magic link email not found: timeout waiting for magic_link email
```
**Solutions**:
- Check that the email was actually sent
- Verify the Gmail address matches the one configured
- Increase timeout: `email_timeout: 60`
- Check Gmail spam folder

### Permission Denied
```
❌ failed to exchange authorization code
```
**Solution**: Copy the complete authorization code from browser, including any trailing characters.

## Advanced Configuration

### Custom Timeout Per User
```yaml
auth_config:
  email_timeout: 60  # Wait 60 seconds instead of default 30
```

### Disable Email Checking
```yaml  
auth_config:
  email_check_enabled: false  # Skip email checking, use manual flow
```

### Environment Variables (CI/CD)
```bash
# For automated environments
export TOD_GMAIL_REFRESH_TOKEN="your_refresh_token"
```

## File Structure

```
tod/
├── internal/email/
│   ├── client.go      # Gmail API wrapper
│   ├── extractor.go   # LLM-powered extraction
│   └── setup.go       # OAuth setup flow
├── .tod/credentials/
│   └── email.json     # Encrypted OAuth credentials
└── examples/
    └── email-auth-users.yaml  # Example configurations
```

## API Reference

### Email Client
```go
client, err := email.NewClient(projectDir)
emails, err := client.GetRecentEmails(30 * time.Second)
```

### Extractor Service  
```go
extractor := email.NewExtractorService(llmClient)
result, err := extractor.ExtractAuthData(emails, email.AuthTypeMagicLink, context)
```

### Auth Flow Manager
```go
authFlow, err := users.NewAuthFlowManager(projectDir, llmClient)
result := authFlow.AuthenticateWithEmailSupport(user)
```
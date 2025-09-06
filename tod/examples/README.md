# Tod Examples

This directory contains example configurations and templates for Tod testing.

## Test Users

### sample-test-users.yaml

This file contains example test user configurations for different authentication types and environments.

**Usage:**
```bash
# Create all users from the sample file
tod users batch --file examples/sample-test-users.yaml

# Copy and customize for your project
cp examples/sample-test-users.yaml my-test-users.yaml
# Edit my-test-users.yaml with your specific users
tod users batch --file my-test-users.yaml
```

### Included User Types:

1. **Admin Users** - Full permission testing
   - Username/password authentication
   - Basic auth for staging

2. **Regular Users** - Standard user flow testing  
   - Form-based login
   - OAuth authentication

3. **API Users** - API endpoint testing
   - Bearer token authentication
   - Environment-specific tokens

4. **Magic Link Users** - Passwordless authentication testing
   - Email-based authentication

### Customization:

1. **Update credentials** - Replace example passwords and tokens with real values
2. **Modify environments** - Change environment names to match your setup
3. **Add custom fields** - Include additional metadata for your specific needs
4. **Adjust auth configs** - Update selectors and endpoints for your application

### Security Notes:

- **Never commit real credentials** to version control
- Use environment variables for sensitive data: `${MY_API_TOKEN}`
- Consider using different credentials for each environment
- Rotate test credentials regularly

## Quick Start:

1. Initialize Tod in your project:
   ```bash
   tod init
   ```

2. Create users from template:
   ```bash
   tod users create --template admin
   ```

3. Or import sample users:
   ```bash
   tod users batch --file examples/sample-test-users.yaml
   ```

4. List your users:
   ```bash
   tod users list
   ```

5. Start testing:
   ```bash
   tod
   ```
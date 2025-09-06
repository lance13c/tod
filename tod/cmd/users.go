package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ciciliostudio/tod/internal/config"
	"github.com/ciciliostudio/tod/internal/ui/components"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// usersCmd represents the users command
var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage test users",
	Long: `Manage test users for your Tod testing environment.

Create, list, and manage test users with different authentication configurations
to streamline your testing workflow.

Examples:
  tod users create              # Interactive user creation
  tod users create --template admin  # Create from template  
  tod users list               # List all configured users
  tod users batch --file users.yaml  # Create multiple users from file`,
}

// createUserCmd creates a new test user
var createUserCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new test user",
	Long: `Create a new test user with interactive setup or from a template.

The user will be configured with authentication details based on your
environment configuration and can be used immediately for testing.`,
	Run: runCreateUser,
}

// listUsersCmd lists all configured test users
var listUsersCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured test users",
	Long:  `Display all configured test users, optionally filtered by environment or auth type.`,
	Run:   runListUsers,
}

// batchUsersCmd creates multiple users from a file
var batchUsersCmd = &cobra.Command{
	Use:   "batch",
	Short: "Create multiple users from file",
	Long: `Create multiple test users from a YAML or JSON configuration file.

The file should contain an array of user definitions with the required fields
for each authentication type.`,
	Run: runBatchUsers,
}

func init() {
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(createUserCmd)
	usersCmd.AddCommand(listUsersCmd)
	usersCmd.AddCommand(batchUsersCmd)

	// create command flags
	createUserCmd.Flags().String("template", "", "Template to use for user creation")
	createUserCmd.Flags().String("env", "", "Environment for the user (defaults to current)")
	createUserCmd.Flags().String("role", "", "User role")
	createUserCmd.Flags().Bool("non-interactive", false, "Skip interactive prompts")

	// list command flags
	listUsersCmd.Flags().String("env", "", "Filter by environment")
	listUsersCmd.Flags().String("auth-type", "", "Filter by authentication type")
	listUsersCmd.Flags().String("role", "", "Filter by user role")
	listUsersCmd.Flags().Bool("show-secrets", false, "Show authentication secrets")

	// batch command flags
	batchUsersCmd.Flags().String("file", "", "File containing user definitions (required)")
	batchUsersCmd.MarkFlagRequired("file")
}

func runCreateUser(cmd *cobra.Command, args []string) {
	// Check if project is initialized
	if todConfig == nil {
		fmt.Println("🚨 Tod is not initialized in this project!")
		fmt.Println("Run 'tod init' to get started.")
		os.Exit(1)
	}

	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	loader := config.NewTestUserLoader(projectDir)
	userConfig, err := loader.Load()
	if err != nil {
		fmt.Printf("❌ Error loading test user config: %v\n", err)
		os.Exit(1)
	}

	templateName, _ := cmd.Flags().GetString("template")
	environment, _ := cmd.Flags().GetString("env")
	role, _ := cmd.Flags().GetString("role")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	// Use current environment if not specified
	if environment == "" {
		environment = todConfig.Current
	}

	// Verify environment exists
	if _, exists := todConfig.Envs[environment]; !exists {
		fmt.Printf("❌ Environment '%s' not found in configuration\n", environment)
		os.Exit(1)
	}

	var user config.TestUser
	if templateName != "" {
		// Create from template
		template, exists := userConfig.GetTemplate(templateName)
		if !exists {
			fmt.Printf("❌ Template '%s' not found\n", templateName)
			fmt.Println("\nAvailable templates:")
			for name, tmpl := range userConfig.Templates {
				fmt.Printf("  • %s: %s\n", name, tmpl.Description)
			}
			os.Exit(1)
		}

		if nonInteractive {
			user = createUserFromTemplateNonInteractive(template, environment, role)
		} else {
			user = createUserFromTemplateInteractive(template, environment, role)
		}
	} else {
		// Interactive creation
		if nonInteractive {
			fmt.Println("❌ Non-interactive mode requires --template flag")
			os.Exit(1)
		}
		user = createUserInteractive(userConfig, environment, role)
	}

	// Add user to configuration
	userConfig.AddUser(user)

	// Save configuration
	err = loader.Save(userConfig)
	if err != nil {
		fmt.Printf("❌ Error saving user configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Test user '%s' created successfully!\n", user.Name)
	fmt.Printf("   • ID: %s\n", user.ID)
	fmt.Printf("   • Environment: %s\n", user.Environment)
	fmt.Printf("   • Auth Type: %s\n", user.AuthType)
	if user.Role != "" {
		fmt.Printf("   • Role: %s\n", user.Role)
	}

	fmt.Printf("\n💡 Use 'tod users list' to see all configured users\n")
	fmt.Printf("🚀 Run 'tod' to start testing with your new user\n")
}

func runListUsers(cmd *cobra.Command, args []string) {
	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	loader := config.NewTestUserLoader(projectDir)
	userConfig, err := loader.Load()
	if err != nil {
		fmt.Printf("❌ Error loading test user config: %v\n", err)
		os.Exit(1)
	}

	// Get filter options
	envFilter, _ := cmd.Flags().GetString("env")
	authTypeFilter, _ := cmd.Flags().GetString("auth-type")
	roleFilter, _ := cmd.Flags().GetString("role")
	showSecrets, _ := cmd.Flags().GetBool("show-secrets")

	if len(userConfig.Users) == 0 {
		fmt.Println("📋 No test users configured yet.")
		fmt.Println("\n💡 Create your first user with: tod users create")
		return
	}

	fmt.Printf("📋 Test Users (%d total)\n\n", len(userConfig.Users))

	for _, user := range userConfig.Users {
		// Apply filters
		if envFilter != "" && user.Environment != envFilter {
			continue
		}
		if authTypeFilter != "" && user.AuthType != authTypeFilter {
			continue
		}
		if roleFilter != "" && user.Role != roleFilter {
			continue
		}

		// Display user info
		fmt.Printf("🧪 %s (%s)\n", user.Name, user.ID)
		fmt.Printf("   Environment: %s\n", user.Environment)
		fmt.Printf("   Auth Type: %s\n", user.AuthType)
		if user.Role != "" {
			fmt.Printf("   Role: %s\n", user.Role)
		}
		if user.Email != "" {
			fmt.Printf("   Email: %s\n", user.Email)
		}
		if user.Username != "" {
			fmt.Printf("   Username: %s\n", user.Username)
		}
		if user.Description != "" {
			fmt.Printf("   Description: %s\n", user.Description)
		}

		// Show auth details if requested
		if showSecrets && user.AuthConfig != nil {
			fmt.Println("   Auth Details:")
			switch user.AuthType {
			case "basic":
				if user.AuthConfig.Username != "" {
					fmt.Printf("     Username: %s\n", user.AuthConfig.Username)
				}
				if user.AuthConfig.Password != "" {
					fmt.Printf("     Password: %s\n", maskSecret(user.AuthConfig.Password))
				}
			case "bearer":
				if user.AuthConfig.Token != "" {
					fmt.Printf("     Token: %s\n", maskSecret(user.AuthConfig.Token))
				}
			case "oauth":
				if user.AuthConfig.Provider != "" {
					fmt.Printf("     Provider: %s\n", user.AuthConfig.Provider)
				}
				if user.AuthConfig.AccessToken != "" {
					fmt.Printf("     Access Token: %s\n", maskSecret(user.AuthConfig.AccessToken))
				}
			}
		}

		fmt.Printf("   Created: %s\n", user.CreatedAt.Format(time.RFC3339))
		fmt.Println()
	}

	if !showSecrets {
		fmt.Println("💡 Use --show-secrets to view authentication details")
	}
}

func runBatchUsers(cmd *cobra.Command, args []string) {
	filePath, _ := cmd.Flags().GetString("file")
	projectDir, _ := cmd.Flags().GetString("project")
	if projectDir == "" {
		projectDir = "."
	}

	// Read and parse batch file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("❌ Error reading file: %v\n", err)
		os.Exit(1)
	}

	var batchConfig struct {
		Users []config.TestUser `yaml:"users"`
	}

	err = yaml.Unmarshal(data, &batchConfig)
	if err != nil {
		fmt.Printf("❌ Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	if len(batchConfig.Users) == 0 {
		fmt.Println("❌ No users found in batch file")
		os.Exit(1)
	}

	// Load existing config
	loader := config.NewTestUserLoader(projectDir)
	userConfig, err := loader.Load()
	if err != nil {
		fmt.Printf("❌ Error loading test user config: %v\n", err)
		os.Exit(1)
	}

	// Add users from batch
	created := 0
	skipped := 0

	for _, user := range batchConfig.Users {
		// Validate required fields
		if user.ID == "" {
			user.ID = generateUserID(user.Name, user.Environment)
		}
		if user.Name == "" {
			fmt.Printf("⚠️  Skipping user with missing name\n")
			skipped++
			continue
		}
		if user.Environment == "" {
			fmt.Printf("⚠️  Skipping user '%s' with missing environment\n", user.Name)
			skipped++
			continue
		}

		// Check if user already exists
		if _, exists := userConfig.GetUser(user.ID); exists {
			fmt.Printf("⚠️  User '%s' already exists, skipping\n", user.Name)
			skipped++
			continue
		}

		userConfig.AddUser(user)
		created++
		fmt.Printf("✅ Created user: %s (%s)\n", user.Name, user.ID)
	}

	// Save configuration
	err = loader.Save(userConfig)
	if err != nil {
		fmt.Printf("❌ Error saving user configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🎉 Batch creation complete: %d created, %d skipped\n", created, skipped)
	fmt.Println("💡 Use 'tod users list' to see all configured users")
}

func createUserInteractive(userConfig *config.TestUserConfig, defaultEnv, defaultRole string) config.TestUser {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🧪 Creating a new test user...")
	fmt.Println()

	// Basic user info
	name := askString(reader, "User name: ", "Test User")
	email := askString(reader, "Email (optional): ", "")
	username := askString(reader, "Username (optional): ", "")

	// Environment
	environment := askString(reader, fmt.Sprintf("Environment [%s]: ", defaultEnv), defaultEnv)

	// Role
	role := askString(reader, "Role (optional): ", defaultRole)

	// Description
	description := askString(reader, "Description (optional): ", "")

	// Auth type selection
	fmt.Println("\n🔐 Choose authentication type:")
	selectedAuth, err := components.RunSelectorWithDetails("Select authentication type:", components.AuthTypeOptions)
	if err != nil {
		fmt.Printf("❌ Auth selection failed: %v\n", err)
		os.Exit(1)
	}

	// Generate user ID
	userID := generateUserID(name, environment)

	// Create base user
	user := config.TestUser{
		ID:          userID,
		Name:        name,
		Email:       email,
		Username:    username,
		Role:        role,
		Description: description,
		Environment: environment,
		AuthType:    selectedAuth.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Configure auth-specific settings
	user.AuthConfig = createAuthConfigInteractive(reader, selectedAuth.ID)

	return user
}

func createUserFromTemplateInteractive(template config.TestUserTemplate, environment, role string) config.TestUser {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("🧪 Creating user from template: %s\n", template.Name)
	fmt.Printf("   %s\n\n", template.Description)

	userData := make(map[string]interface{})

	// Collect field values
	for _, field := range template.Fields {
		var value string
		prompt := fmt.Sprintf("%s: ", field.Label)
		if field.Required {
			prompt = fmt.Sprintf("%s (required): ", field.Label)
		}

		if field.Type == "select" {
			// Show options for select fields
			fmt.Printf("%s\n", field.Label)
			for i, option := range field.Options {
				fmt.Printf("  %d) %s\n", i+1, option)
			}
			choice := askChoice(reader, "Select option: ", 1, len(field.Options))
			value = field.Options[choice-1]
		} else {
			value = askString(reader, prompt, field.Default)
		}

		if field.Required && value == "" {
			fmt.Printf("❌ %s is required\n", field.Label)
			os.Exit(1)
		}

		userData[field.Name] = value
	}

	// Override with provided values
	if environment != "" {
		userData["environment"] = environment
	}
	if role != "" {
		userData["role"] = role
	}

	// Generate user ID
	userName := getStringValue(userData, "name", "Test User")
	userID := generateUserID(userName, getStringValue(userData, "environment", environment))

	// Create user from collected data
	user := config.TestUser{
		ID:          userID,
		Name:        userName,
		Email:       getStringValue(userData, "email", ""),
		Username:    getStringValue(userData, "username", ""),
		Role:        getStringValue(userData, "role", template.Role),
		Description: template.Description,
		Environment: getStringValue(userData, "environment", environment),
		AuthType:    template.AuthType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create auth config from template data
	user.AuthConfig = createAuthConfigFromData(template.AuthType, userData)

	return user
}

func createUserFromTemplateNonInteractive(template config.TestUserTemplate, environment, role string) config.TestUser {
	// Use defaults from template
	userData := template.Defaults
	if userData == nil {
		userData = make(map[string]interface{})
	}

	// Override with provided values
	if environment != "" {
		userData["environment"] = environment
	}
	if role != "" {
		userData["role"] = role
	}

	// Generate user ID
	userName := getStringValue(userData, "name", template.Name+" User")
	userID := generateUserID(userName, getStringValue(userData, "environment", environment))

	// Create user
	user := config.TestUser{
		ID:          userID,
		Name:        userName,
		Email:       getStringValue(userData, "email", ""),
		Username:    getStringValue(userData, "username", ""),
		Role:        getStringValue(userData, "role", template.Role),
		Description: template.Description,
		Environment: getStringValue(userData, "environment", environment),
		AuthType:    template.AuthType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create auth config from template data
	user.AuthConfig = createAuthConfigFromData(template.AuthType, userData)

	return user
}

func createAuthConfigInteractive(reader *bufio.Reader, authType string) *config.TestUserAuthConfig {
	authConfig := &config.TestUserAuthConfig{}

	fmt.Printf("\n🔐 Configuring %s authentication:\n", authType)

	switch authType {
	case "none":
		// No configuration needed
		return nil

	case "basic":
		authConfig.Username = askString(reader, "Username: ", "")
		authConfig.Password = askString(reader, "Password: ", "")

	case "bearer":
		authConfig.Token = askString(reader, "Bearer token: ", "")

	case "oauth":
		// Select OAuth provider
		fmt.Println("\n🔑 Choose OAuth provider:")
		selectedProvider, err := components.RunSelectorWithDetails("Select OAuth provider:", components.OAuthProviderOptions)
		if err != nil {
			fmt.Printf("❌ OAuth provider selection failed: %v\n", err)
			return authConfig
		}

		authConfig.Provider = selectedProvider.ID
		authConfig.ClientID = askString(reader, "Client ID: ", "")
		authConfig.ClientSecret = askString(reader, "Client Secret: ", "")

	case "magic_link":
		authConfig.EmailEndpoint = askString(reader, "Magic link endpoint: ", "/auth/magic-link")

	case "username_password":
		authConfig.LoginFormURL = askString(reader, "Login form URL: ", "/login")
		authConfig.UsernameField = askString(reader, "Username field selector: ", "#email")
		authConfig.PasswordField = askString(reader, "Password field selector: ", "#password")
		authConfig.SubmitButton = askString(reader, "Submit button selector: ", "button[type=\"submit\"]")
	}

	return authConfig
}

func createAuthConfigFromData(authType string, data map[string]interface{}) *config.TestUserAuthConfig {
	if authType == "none" {
		return nil
	}

	authConfig := &config.TestUserAuthConfig{}

	switch authType {
	case "basic":
		authConfig.Username = getStringValue(data, "username", "")
		authConfig.Password = getStringValue(data, "password", "")

	case "bearer":
		authConfig.Token = getStringValue(data, "token", "")

	case "oauth":
		authConfig.Provider = getStringValue(data, "provider", "")
		authConfig.ClientID = getStringValue(data, "client_id", "")
		authConfig.ClientSecret = getStringValue(data, "client_secret", "")

	case "magic_link":
		authConfig.EmailEndpoint = getStringValue(data, "email_endpoint", "/auth/magic-link")

	case "username_password":
		authConfig.LoginFormURL = getStringValue(data, "login_form_url", "/login")
		authConfig.UsernameField = getStringValue(data, "username_field", "#email")
		authConfig.PasswordField = getStringValue(data, "password_field", "#password")
		authConfig.SubmitButton = getStringValue(data, "submit_button", "button[type=\"submit\"]")
	}

	return authConfig
}

// Helper functions

func generateUserID(name, environment string) string {
	base := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	if environment != "" {
		base = base + "_" + environment
	}
	return base + "_" + fmt.Sprintf("%d", time.Now().Unix())
}

func getStringValue(data map[string]interface{}, key, defaultValue string) string {
	if value, exists := data[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}
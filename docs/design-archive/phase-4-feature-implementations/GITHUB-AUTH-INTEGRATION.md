# GitHub Authentication Integration

## Overview

tfpipboy now automatically detects and injects GitHub CLI authentication into Terraform executions, enabling seamless use of the Terraform GitHub provider without manual token management.

## Features

### Automatic Token Injection

When you run Terraform modules that use the GitHub provider, tfpipboy automatically:
1. Detects your local `gh` CLI authentication
2. Retrieves the active GitHub token
3. Injects it as `GITHUB_TOKEN` environment variable into the Terraform execution

### Multiple Account Support

If you have multiple GitHub accounts authenticated via `gh` CLI:
- tfpipboy detects all available accounts
- Shows the active account in the execution plan preview
- Automatically uses the active account's token
- Displays account count in the preview

### Execution Plan Preview

The execution plan preview now shows GitHub authentication status:

```
Authentication Status:
  ✓ Azure:   my-subscription-name
  ✓ AWS:     123456789012
  ✓ GitHub:  StanleyXie (active)
```

With multiple accounts:
```
Authentication Status:
  ✓ GitHub:  StanleyXie (active: StanleyXie, available: 2 accounts)
```

## How It Works

### 1. GitHub CLI Authentication Check

tfpipboy runs `gh auth status` to detect:
- Authenticated GitHub accounts
- Active account
- Token availability

### 2. Token Retrieval

For the active account, tfpipboy runs `gh auth token` to retrieve the OAuth token.

### 3. Environment Variable Injection

The token is injected as `GITHUB_TOKEN` into the Terraform workspace environment:

```bash
export GITHUB_TOKEN="gho_xxxxxxxxxxxxxxxxxxxx"
terraform apply ...
```

### 4. Terraform GitHub Provider Usage

The Terraform GitHub provider automatically uses the `GITHUB_TOKEN` environment variable:

```hcl
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 5.0"
    }
  }
}

provider "github" {
  # Token is automatically picked up from GITHUB_TOKEN environment variable
  # No need to specify token explicitly
}

resource "github_repository" "example" {
  name        = "example-repo"
  description = "My example repository"
  visibility  = "private"
}
```

## Prerequisites

### Install GitHub CLI

```bash
# macOS
brew install gh

# Linux
sudo apt install gh

# Windows
winget install --id GitHub.cli
```

### Authenticate with GitHub

```bash
gh auth login
```

Follow the prompts to authenticate with your GitHub account.

### Verify Authentication

```bash
gh auth status
```

Output:
```
github.com
  ✓ Logged in to github.com account YourUsername (keyring)
  - Active account: true
  - Git operations protocol: https
  - Token: gho_************************************
  - Token scopes: 'gist', 'read:org', 'repo', 'workflow'
```

## Usage Examples

### Example 1: Basic GitHub Resource Management

**Terraform Configuration:**
```hcl
# main.tf
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 5.0"
    }
  }
}

provider "github" {
  # Token automatically from GITHUB_TOKEN env var
}

resource "github_repository" "my_repo" {
  name        = "my-terraform-repo"
  description = "Created via tfpipboy"
  visibility  = "private"
  
  has_issues = true
  has_wiki   = true
}
```

**tfpipboy Configuration:**
```yaml
# .tfpipboy/tfproject.yaml
version: "1.0"

modules:
  github-resources:
    path: "./modules/github"
    backend:
      type: "local"
      path: "terraform.tfstate"
```

**Run with tfpipboy:**
```bash
tfpipboy orchestrate plan
# ✓ GitHub: YourUsername (active)
# Terraform will use your gh CLI token automatically

tfpipboy orchestrate apply
```

### Example 2: Multiple Environments with Different GitHub Accounts

**Switch GitHub Account:**
```bash
# List accounts
gh auth status

# Switch to different account
gh auth switch

# Verify active account
gh auth status
```

**Run tfpipboy:**
```bash
tfpipboy orchestrate plan
# ✓ GitHub: DifferentAccount (active: DifferentAccount, available: 2 accounts)
```

The newly selected account's token will be used automatically.

### Example 3: Manual Token Override

If you need to override the automatic token (e.g., for CI/CD):

```yaml
# .tfpipboy/tfproject.yaml
modules:
  github-resources:
    path: "./modules/github"
    variables:
      GITHUB_TOKEN: "${env:CUSTOM_GITHUB_TOKEN}"
```

Or set environment variable before running:
```bash
export GITHUB_TOKEN="ghp_custom_token_here"
tfpipboy orchestrate apply
```

tfpipboy respects manually set `GITHUB_TOKEN` and won't override it.

## Security Considerations

### Token Security

1. **Keychain Storage**: `gh` CLI stores tokens securely in your system keychain
2. **No Token Logging**: tfpipboy never logs the actual token value
3. **Workspace Isolation**: Tokens are only passed to isolated Terraform workspaces
4. **Process Lifetime**: Tokens exist only during the Terraform execution

### Token Scopes

Ensure your GitHub token has appropriate scopes:

```bash
gh auth refresh -s repo -s workflow -s admin:org
```

Common scopes needed:
- `repo` - Full control of repositories
- `workflow` - Update GitHub Actions workflows
- `admin:org` - Full control of organizations
- `delete_repo` - Delete repositories

### Best Practices

1. **Use Personal Access Tokens (PAT) for CI/CD**:
   ```bash
   # In CI/CD pipelines
   export GITHUB_TOKEN="${{ secrets.GITHUB_TOKEN }}"
   ```

2. **Regularly Rotate Tokens**:
   ```bash
   gh auth refresh
   ```

3. **Use Minimal Scopes**: Only grant necessary permissions

4. **Separate Accounts for Different Environments**:
   - Development account for dev/test
   - Production account for prod deployments

## Troubleshooting

### GitHub Token Not Found

**Error:**
```
GitHub authentication not detected
```

**Solution:**
```bash
gh auth login
gh auth status  # Verify authentication
```

### Invalid Token Scopes

**Error:**
```
Error: GET https://api.github.com/user: 403 Resource not accessible by personal access token
```

**Solution:**
```bash
gh auth refresh -s repo -s workflow
```

### Multiple Accounts - Wrong Account Selected

**Check active account:**
```bash
gh auth status
```

**Switch account:**
```bash
gh auth switch
```

**Manually select account:**
```bash
gh auth login  # Login with specific account
```

### Token Not Injected

**Debug:**
```bash
# Check if token is available
gh auth token

# Check workspace environment
# Token should appear in Terraform execution
```

If token is not being injected, verify:
1. `gh` CLI is installed and in PATH
2. You're authenticated: `gh auth status`
3. Token can be retrieved: `gh auth token`

## Implementation Details

### Code Components

1. **auth.go:143** - `CheckGitHubAuth()` function
   - Detects GitHub CLI authentication
   - Parses multiple accounts
   - Retrieves active account and token

2. **workspace.go:515** - `injectGitHubToken()` function
   - Checks if `GITHUB_TOKEN` already exists
   - Retrieves token from `gh auth token`
   - Injects into workspace environment

3. **display.go:513** - Enhanced authentication display
   - Shows active account
   - Displays multiple account count
   - Indicates which account is active

### Environment Variable Precedence

1. **Highest Priority**: Manually set `GITHUB_TOKEN` in environment
2. **Medium Priority**: Module/instance-level `variables` with `GITHUB_TOKEN`
3. **Lowest Priority**: Auto-injected from `gh` CLI (default)

This ensures manual overrides always take precedence.

### Error Handling

- GitHub authentication failures are non-fatal
- Workspace creation continues even if token injection fails
- Terraform will fail at runtime if it needs GitHub auth but token is missing

## Future Enhancements

### Planned Features

1. **Account Selection Prompt**:
   - Interactive prompt to select GitHub account when multiple are available
   - Configurable default account per module/instance

2. **Token Caching**:
   - Cache token retrieval to avoid repeated `gh auth token` calls
   - Respect token TTL

3. **GitHub Enterprise Support**:
   - Support for `gh` authentication with GitHub Enterprise Server
   - Multiple host support (`github.com`, `github.enterprise.com`)

4. **Token Validation**:
   - Verify token scopes before execution
   - Warn if required scopes are missing

## Related Documentation

- [Terraform GitHub Provider](https://registry.terraform.io/providers/integrations/github/latest/docs)
- [GitHub CLI Authentication](https://cli.github.com/manual/gh_auth)
- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)

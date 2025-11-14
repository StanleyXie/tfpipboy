# Terraform Setup Guide

## Why "terraform: command not found"?

tf-pipboy is a **wrapper** around Terraform, which means Terraform itself needs to be installed on your system separately.

## Installing Terraform on Mac

### Option 1: Homebrew (Recommended)

```bash
brew tap hashicorp/tap
brew install hashicorp/tap/terraform
```

Or simply:
```bash
brew install terraform
```

### Option 2: Direct Download

1. Visit https://www.terraform.io/downloads
2. Download the appropriate package for your OS
3. Unzip and move to PATH:
```bash
unzip terraform_*_darwin_*.zip
sudo mv terraform /usr/local/bin/
```

### Verify Installation

```bash
terraform version
```

You should see output like:
```
Terraform v1.x.x
on darwin_arm64
```

## Using Terraform with tf-pipboy

Once Terraform is installed, you can run it through tf-pipboy:

```bash
./bin/tfpipboy

# Then type any terraform command:
> terraform version
> terraform init
> terraform workspace list
> terraform plan
```

## Other Tools You Might Need

tf-pipboy also detects authentication for these CLIs (install as needed):

### Azure CLI
```bash
brew install azure-cli
az login
```

### GitHub CLI
```bash
brew install gh
gh auth login
```

### AWS CLI
```bash
brew install awscli
aws configure
```

### Google Cloud SDK
```bash
brew install google-cloud-sdk
gcloud auth login
```

## What tf-pipboy Provides

tf-pipboy **wraps** Terraform to add:
- ✅ Real-time context awareness (workspace, backend, module)
- ✅ Authentication status monitoring (Azure, GitHub, AWS, GCP)
- ✅ Environment variable detection
- ✅ Enhanced command history and completion
- ✅ Professional TUI with Emacs/Readline keybindings

## Quick Start

1. Install Terraform (see above)
2. Run tf-pipboy:
   ```bash
   cd /path/to/your/terraform/project
   ./bin/tfpipboy
   ```
3. Check status bar for context and auth status
4. Run Terraform commands as normal

## Troubleshooting

### Command not found errors

If you see:
```
Error: exec: "terraform": executable file not found in $PATH
Hint: 'terraform' command not found in PATH. Install it first:
  brew install terraform
```

**Solution:** Install Terraform using one of the methods above.

### PATH issues

If Terraform is installed but tf-pipboy can't find it:

1. Check where Terraform is installed:
   ```bash
   which terraform
   ```

2. Check your PATH:
   ```bash
   echo $PATH
   ```

3. Ensure Terraform's location is in your PATH. Add to `~/.zshrc` or `~/.bash_profile`:
   ```bash
   export PATH="/usr/local/bin:$PATH"
   # Or wherever terraform is installed
   ```

4. Reload your shell:
   ```bash
   source ~/.zshrc  # or source ~/.bash_profile
   ```

### Shell integration

tf-pipboy executes commands through your shell (`$SHELL`), so:
- ✅ All your shell aliases work
- ✅ All your environment variables are inherited
- ✅ Your PATH is preserved

If commands work in your terminal but not in tf-pipboy, check:
```bash
echo $SHELL  # Should show /bin/zsh or /bin/bash
```

## Example Workflow

```bash
# 1. Install Terraform
brew install terraform

# 2. Navigate to your Terraform project
cd ~/projects/terraform/my-infrastructure

# 3. Run tf-pipboy
./bin/tfpipboy

# 4. Check context in status bar
# Status bar shows: TF[WS:default Backend:S3] | Auth[✓GH:user ✗Az]

# 5. Run Terraform commands
> terraform init
> terraform workspace new production
> terraform workspace select production

# Notice status bar updates: TF[WS:production Backend:S3]

> terraform plan
> terraform apply
```

## Need Help?

- Terraform installation issues: https://terraform.io/downloads
- tf-pipboy issues: https://github.com/StanleyXie/tf-pipboy/issues
- Terraform documentation: https://terraform.io/docs

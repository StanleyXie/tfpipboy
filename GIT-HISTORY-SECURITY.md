# Git History Security Guide

## ⚠️ Important Security Consideration

Even though sensitive files (examples, tests) have been removed from the current working tree, **they still exist in git history**. Before making this repository public, you must address this security risk.

---

## Current Situation

### Files Removed from Working Tree
- `examples/` directory (56 files) - Removed in commit `a39d04f`
- `tests/` directory (28 files) - Removed in commit `ca94d85`
- `.tfpipboy/` directory - Moved outside repo

### Security Risk
These files and their contents are **still accessible** in git history:
```bash
# Anyone can access removed files
git checkout a39d04f~1 -- examples/
git checkout ca94d85~1 -- tests/
```

### What Might Be in History
- Azure subscription IDs, tenant IDs
- AWS account IDs
- GCP project IDs  
- Backend configuration with resource names
- IP addresses, hostnames
- Email addresses
- Company-specific information
- Test data with real infrastructure details

---

## Solutions

You have **three main options** depending on your situation:

### Option 1: Start Fresh (Recommended for High Security)

**Best for**: Maximum security, when history is not valuable

**Process**:
1. Export current clean state
2. Create new repository
3. Import clean state as initial commit

**Steps**:
```bash
# 1. Create archive of current clean state
cd /Users/stanleyxie/Workspace/Projects/tf-pipboy
tar -czf ../tf-pipboy-clean.tar.gz \
  --exclude='.git' \
  --exclude='examples' \
  --exclude='.tfpipboy' \
  .

# 2. Create new repository
cd ..
mkdir tf-pipboy-clean
cd tf-pipboy-clean
git init
git checkout -b main

# 3. Extract clean code
tar -xzf ../tf-pipboy-clean.tar.gz

# 4. Initial commit
git add .
git commit -m "Initial public release v0.6.0

tf-pipboy - Terraform orchestration tool
- Parallel execution with dependency management
- Authentication monitoring
- Live board TUI
- Complete documentation"

# 5. Add remote and push
git remote add origin https://github.com/StanleyXie/tf-pipboy.git
git push -u origin main --force

# 6. Create release tag
git tag -a v0.6.0 -m "Release v0.6.0 - Initial public release"
git push origin v0.6.0
```

**Pros**:
- ✅ Complete removal of all sensitive data
- ✅ Clean git history
- ✅ No chance of leaks
- ✅ Simple and certain

**Cons**:
- ❌ Loses all development history
- ❌ Loses commit attribution
- ❌ Need to update existing clones

---

### Option 2: Rewrite History (Preserve Recent History)

**Best for**: When you want to keep recent development history but remove sensitive files

**Tools**: BFG Repo-Cleaner or git filter-repo

#### Using BFG Repo-Cleaner

```bash
# 1. Install BFG
brew install bfg

# 2. Create backup
cd /Users/stanleyxie/Workspace/Projects
git clone --mirror tf-pipboy tf-pipboy-backup.git

# 3. Clone fresh copy for cleaning
git clone tf-pipboy tf-pipboy-clean
cd tf-pipboy-clean

# 4. Delete sensitive files/folders from history
bfg --delete-folders examples
bfg --delete-folders tests  
bfg --delete-folders .tfpipboy

# 5. Clean up
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# 6. Verify clean
git log --all --oneline
git log --all --pretty=format: --name-only | sort -u | grep -E "(examples|tests|\.tfpipboy)"

# 7. Force push (DESTRUCTIVE!)
git push origin --force --all
git push origin --force --tags
```

#### Using git filter-repo (More Modern)

```bash
# 1. Install git filter-repo
brew install git-filter-repo

# 2. Clone fresh copy
cd /Users/stanleyxie/Workspace/Projects
git clone tf-pipboy tf-pipboy-clean
cd tf-pipboy-clean

# 3. Remove sensitive paths
git filter-repo --path examples --invert-paths
git filter-repo --path tests --invert-paths
git filter-repo --path .tfpipboy --invert-paths

# 4. Add remote back (filter-repo removes it)
git remote add origin https://github.com/StanleyXie/tf-pipboy.git

# 5. Force push (DESTRUCTIVE!)
git push origin --force --all
git push origin --force --tags
```

**Pros**:
- ✅ Keeps development history
- ✅ Preserves commit messages and attribution
- ✅ Removes sensitive files completely

**Cons**:
- ❌ Requires force push (breaks existing clones)
- ❌ Complex process
- ❌ Risk of mistakes
- ❌ All contributors need to re-clone

---

### Option 3: Squash History (Keep Attribution, Clean History)

**Best for**: When you want a middle ground - some history but simplified

**Process**:
```bash
# 1. Create orphan branch (no history)
cd /Users/stanleyxie/Workspace/Projects/tf-pipboy
git checkout --orphan clean-main

# 2. Add all current clean files
git add .

# 3. Create single commit
git commit -m "Initial public release v0.6.0

Complete Terraform orchestration tool with:
- Parallel execution with dependency management
- Authentication monitoring (AWS, Azure, GCP, GitHub)
- Live board TUI with real-time status
- Comprehensive documentation

Development history: October 2024 - November 2024
Previous commits squashed for security

Contributors:
- Stanley Xie <stanley@example.com>

Changelog: See CHANGELOG.md for detailed history"

# 4. Replace main branch
git branch -D main
git branch -m main

# 5. Force push
git push origin main --force

# 6. Create tag
git tag -a v0.6.0 -m "Release v0.6.0 - Initial public release"
git push origin v0.6.0 --force
```

**Pros**:
- ✅ Clean, single-commit history
- ✅ No sensitive data
- ✅ Can credit contributors in commit message
- ✅ Simple process

**Cons**:
- ❌ Loses detailed development history
- ❌ Requires force push

---

## Recommended Approach

### For tf-pipboy: **Option 1 (Start Fresh)** 

**Reasoning**:
1. Repository contains real infrastructure examples
2. Test files may have actual backend configurations
3. Clean break is safest
4. History value is low (pre-public development)
5. v0.6.0 is good starting point

### Implementation Steps

```bash
# Step 1: Verify current state is clean
cd /Users/stanleyxie/Workspace/Projects/tf-pipboy
git status  # Should be clean

# Step 2: Create clean archive
tar -czf ../tf-pipboy-clean-$(date +%Y%m%d).tar.gz \
  --exclude='.git' \
  --exclude='examples' \
  --exclude='.tfpipboy' \
  --exclude='*.log' \
  --exclude='.DS_Store' \
  .

# Step 3: Verify archive
tar -tzf ../tf-pipboy-clean-$(date +%Y%m%d).tar.gz | head -20

# Step 4: Create new clean repository
cd ..
mkdir tf-pipboy-public
cd tf-pipboy-public
git init -b main

# Step 5: Extract clean code
tar -xzf ../tf-pipboy-clean-$(date +%Y%m%d).tar.gz

# Step 6: Create .gitignore
cat > .gitignore << 'EOF'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
tfpipboy
bin/
dist/

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/
node_modules/

# Go workspace file
go.work

# Python
__pycache__/
*.py[cod]
*$py.class
.Python
venv/
env/
ENV/
*.egg-info/
dist/
build/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~
.DS_Store

# Claude Code
.CLAUDE/

# tf-pipboy specific
/.tfpipboy/
*.tfpipboy.cache

# Terraform
.terraform/
*.tfstate
*.tfstate.*
.terraform.lock.hcl

# Temporary files
*.log
*.tmp

# Examples and tests (keep out of public repo)
examples/
tests/
EOF

# Step 7: Initial commit
git add .
git commit -m "feat: initial public release v0.6.0

Terraform orchestration tool with parallel execution and real-time monitoring.

Features:
- Parallel execution with dependency management
- Authentication monitoring (AWS, Azure, GCP, GitHub)
- Live board TUI with real-time status
- Configuration validation
- Pipeline support
- Isolated workspace management

Technology stack:
- Go 1.21+
- Cobra (CLI framework)
- Bubble Tea (TUI framework)

Documentation:
- Complete user guides
- Configuration reference
- Getting started guide
- Security policy
- Project roadmap"

# Step 8: Create version tag
git tag -a v0.6.0 -m "Release v0.6.0 - Initial public release

Features:
- Comprehensive configuration validation with error aggregation
- Authentication monitoring and blocking
- Operation-specific timeout handling
- Live board TUI
- Parallel execution engine

See CHANGELOG.md for complete details."

# Step 9: Add remote (when ready to make public)
git remote add origin https://github.com/StanleyXie/tf-pipboy.git

# Step 10: Push (when ready)
# git push -u origin main --force
# git push origin v0.6.0 --force
```

---

## Before Making Repository Public

### Pre-Push Checklist

- [ ] Verify no sensitive data in working tree
  ```bash
  find . -name "*.tfvars" -o -name "*.hcl" | grep -v ".git"
  find . -type f -name "*secret*" -o -name "*password*" | grep -v ".git"
  ```

- [ ] Verify .gitignore excludes sensitive paths
  ```bash
  cat .gitignore | grep -E "(examples|tests|\.tfpipboy)"
  ```

- [ ] Run security scans
  ```bash
  # Install gitleaks
  brew install gitleaks
  
  # Scan for secrets
  gitleaks detect --source . --verbose
  ```

- [ ] Verify no credentials in git history
  ```bash
  git log --all --pretty=format: --name-only | sort -u | grep -E "(secret|password|key|token|credential)"
  ```

- [ ] Check file sizes (large files = potential data leaks)
  ```bash
  git ls-tree -r -l --full-name HEAD | sort -k 4 -n | tail -20
  ```

- [ ] Review all documentation
  - [ ] No company-specific URLs
  - [ ] No internal hostnames
  - [ ] No real subscription IDs
  - [ ] No email addresses (except public contact)

### Post-Push Security

After making public, you **cannot** remove data from public git history. Once pushed:
- Data is permanently public
- Anyone can clone and keep copies
- Rewriting history doesn't help (copies exist)

**Therefore**: Get it right before pushing!

---

## Additional Security Measures

### 1. Enable GitHub Security Features

When repository is public:
```bash
# Via GitHub Settings:
# - Enable Dependabot alerts
# - Enable Secret scanning
# - Enable Push protection
# - Configure branch protection rules
```

### 2. Monitor for Leaks

```bash
# Set up monitoring
git config --global alias.check-secrets '!gitleaks detect --source . --verbose'

# Regular checks
git check-secrets
```

### 3. Contributor Guidelines

Update CONTRIBUTING.md:
```markdown
## Security Guidelines

Never commit:
- Real credentials or tokens
- Actual subscription/account IDs
- Production URLs or hostnames
- Personal email addresses
- Test data from real infrastructure

Always:
- Use placeholders (e.g., `xxx-xxx-xxx` for IDs)
- Review changes before committing
- Run `gitleaks detect` before push
```

---

## Emergency: If Secrets Are Already Public

If you accidentally push secrets:

### Immediate Actions (First 5 minutes)

1. **Rotate ALL credentials immediately**
   ```bash
   # AWS
   aws iam delete-access-key --access-key-id AKIAXXXXXXX
   
   # Azure
   az ad sp credential reset --id <app-id>
   
   # GitHub
   # Revoke token at: https://github.com/settings/tokens
   ```

2. **Delete repository** (if just pushed)
   ```bash
   # On GitHub: Settings → Danger Zone → Delete repository
   ```

3. **Contact GitHub Support**
   - Request cache purge
   - Request removal from search indexes

### Longer Term Actions

1. **Audit access logs**
2. **Check for unauthorized access**
3. **Update security procedures**
4. **Incident report and lessons learned**

---

## Summary

**Recommended for tf-pipboy**: Start fresh with clean repository

**Why**: Maximum security, simple process, appropriate for public release

**When to execute**: After completing security review from SECURITY-PRE-RELEASE-CHECKLIST.md

**Before making public**:
1. ✅ Create clean repository (Option 1 above)
2. ✅ Run security scans (gitleaks)
3. ✅ Review all documentation
4. ✅ Complete security checklist
5. ✅ Test installation and basic functionality
6. ✅ Final review
7. 🚀 Make repository public

---

**Questions?** Review this guide carefully and test the process with a test repository first.

**Last Updated**: November 2024

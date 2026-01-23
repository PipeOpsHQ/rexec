# SDK Publishing Guide

This document describes how to set up and use the SDK publishing workflow.

## GitHub Actions Workflow

The `publish-sdks.yml` workflow automates publishing all SDKs to their respective package registries.

### Triggers

1. **On Release**: Automatically triggered when a new GitHub release is published
2. **Manual Dispatch**: Can be triggered manually with custom options

### Manual Trigger Options

- **version**: Version to publish (e.g., `1.0.0`)
- **sdks**: Comma-separated list of SDKs to publish (e.g., `js,python`) or `all`
- **dry_run**: If true, validates packages without publishing

## Required Secrets

Configure these secrets in your repository settings (`Settings → Secrets and variables → Actions`):

### npm (JavaScript/TypeScript)

| Secret | Description |
|--------|-------------|
| `NPM_TOKEN` | npm automation token with publish access |

**How to get**: 
1. Go to https://www.npmjs.com/settings/tokens
2. Create new "Automation" token
3. Ensure the token has publish access to `@pipeopshq` scope

### PyPI (Python)

| Secret | Description |
|--------|-------------|
| `PYPI_TOKEN` | PyPI API token |

**How to get**:
1. Go to https://pypi.org/manage/account/token/
2. Create new API token with "Upload packages" scope
3. Scope can be project-specific for `rexec` package

### crates.io (Rust)

| Secret | Description |
|--------|-------------|
| `CRATES_IO_TOKEN` | crates.io API token |

**How to get**:
1. Go to https://crates.io/settings/tokens
2. Create new token with publish scope

### RubyGems (Ruby)

| Secret | Description |
|--------|-------------|
| `RUBYGEMS_API_KEY` | RubyGems API key |

**How to get**:
1. Go to https://rubygems.org/profile/api_keys
2. Create new API key with "Push rubygems" scope

### Maven Central (Java)

| Secret | Description |
|--------|-------------|
| `OSSRH_USERNAME` | Sonatype OSSRH username |
| `OSSRH_TOKEN` | Sonatype OSSRH token |
| `GPG_PRIVATE_KEY` | GPG private key for signing |
| `GPG_PASSPHRASE` | GPG key passphrase |

**How to get**:
1. Register at https://issues.sonatype.org/
2. Create JIRA ticket to claim `io.pipeops` namespace
3. Generate GPG key: `gpg --full-generate-key`
4. Export private key: `gpg --export-secret-keys --armor YOUR_KEY_ID`
5. Publish public key: `gpg --keyserver keyserver.ubuntu.com --send-keys YOUR_KEY_ID`

### NuGet (.NET)

| Secret | Description |
|--------|-------------|
| `NUGET_API_KEY` | NuGet.org API key |

**How to get**:
1. Go to https://www.nuget.org/account/apikeys
2. Create new API key with push scope for `Rexec` package

### Packagist (PHP)

PHP packages are automatically updated via GitHub webhook. No secrets required.

**Setup**:
1. Go to https://packagist.org/
2. Submit package: https://github.com/PipeOpsHQ/rexec
3. Enable GitHub Service Hook for automatic updates

## SDK Package Registry URLs

| SDK | Registry | Package Name |
|-----|----------|--------------|
| JavaScript | [npm](https://www.npmjs.com/package/@pipeopshq/rexec) | `@pipeopshq/rexec` |
| Python | [PyPI](https://pypi.org/project/rexec/) | `rexec` |
| Rust | [crates.io](https://crates.io/crates/rexec) | `rexec` |
| Ruby | [RubyGems](https://rubygems.org/gems/rexec) | `rexec` |
| Java | [Maven Central](https://central.sonatype.com/artifact/io.pipeops/rexec) | `io.pipeops:rexec` |
| .NET | [NuGet](https://www.nuget.org/packages/Rexec) | `Rexec` |
| PHP | [Packagist](https://packagist.org/packages/pipeopshq/rexec) | `pipeopshq/rexec` |
| Go | GitHub | `github.com/PipeOpsHQ/rexec-go` |

## Version Management

The workflow automatically:
1. Extracts version from release tag (removes `v` prefix)
2. Updates version in each SDK's package manifest
3. Builds and publishes the package

### Manual Version Bump

To manually update versions across all SDKs:

```bash
# JavaScript
cd sdk/js && npm version 1.2.3 --no-git-tag-version

# Python
sed -i 's/version = ".*"/version = "1.2.3"/' sdk/python/pyproject.toml

# Rust
sed -i 's/^version = ".*"/version = "1.2.3"/' sdk/rust/Cargo.toml

# Ruby
sed -i 's/spec.version = ".*"/spec.version = "1.2.3"/' sdk/ruby/rexec.gemspec

# Java
cd sdk/java && mvn versions:set -DnewVersion=1.2.3

# .NET
sed -i 's/<Version>.*<\/Version>/<Version>1.2.3<\/Version>/' sdk/dotnet/Rexec.csproj

# PHP (version in composer.json is optional for libraries)
```

## Dry Run

To test publishing without actually releasing:

1. Go to Actions → Publish SDKs → Run workflow
2. Enter version (e.g., `1.0.0`)
3. Select SDKs to test
4. Check "Dry run" checkbox
5. Run workflow

The workflow will validate packages and show what would be published.

## Troubleshooting

### npm: 403 Forbidden
- Ensure `NPM_TOKEN` has publish access
- Check if package name is available or you own the scope

### PyPI: 400 Bad Request
- Version may already exist on PyPI
- Check package name conflicts

### crates.io: Unauthorized
- Regenerate token at crates.io
- Ensure crate name is available

### Maven Central: Signature Failed
- Verify GPG key is correctly exported
- Check passphrase is correct
- Ensure public key is published to keyserver

### NuGet: API Key Invalid
- Regenerate API key
- Ensure key has push permissions for package

## Go SDK Note

The Go SDK doesn't require registry publishing. Go modules are fetched directly from the Git repository. When a new version is tagged:

1. Users can reference it: `go get github.com/PipeOpsHQ/rexec-go@v1.2.3`
2. The Go module proxy caches it automatically
3. No additional publishing steps needed

To verify Go module availability:
```bash
go list -m github.com/PipeOpsHQ/rexec-go@v1.2.3
```

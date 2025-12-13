# Release Process

This repository uses GitHub Actions to automatically build and publish releases.

## Creating a Release

To create a new release, simply create and push a git tag:

```bash
# Create a new tag (e.g., v1.0.0)
git tag v1.0.0

# Push the tag to GitHub
git push origin v1.0.0
```

## What Gets Built

When you push a tag, the release workflow automatically:

1. **Builds cross-platform binaries**:
   - `linux/amd64`
   - `linux/arm64`

2. **Creates release artifacts**:
   - Compressed `.tar.gz` archives for each platform
   - SHA256 checksums for verification
   - Combined `checksums.txt` file

3. **Builds and publishes Docker image**:
   - Multi-architecture support (linux/amd64, linux/arm64)
   - Published to GitHub Container Registry (ghcr.io)
   - Tagged with version, major.minor, major, and latest

4. **Creates GitHub Release**:
   - Release notes with installation instructions
   - All binary artifacts attached
   - Links to Docker images

## Docker Image Tags

Each release creates multiple Docker image tags:

- `ghcr.io/{owner}/traefik-config-middleware:1.0.0` (full version)
- `ghcr.io/{owner}/traefik-config-middleware:1.0` (major.minor)
- `ghcr.io/{owner}/traefik-config-middleware:1` (major)
- `ghcr.io/{owner}/traefik-config-middleware:latest`

## Required Repository Settings

Ensure the following GitHub repository settings are configured:

### 1. Workflow Permissions

Go to **Settings** → **Actions** → **General** → **Workflow permissions**:
- Enable: **"Read and write permissions"**
- Enable: **"Allow GitHub Actions to create and approve pull requests"** (optional)

### 2. Package Visibility

After the first release, go to **Packages** and ensure the package visibility is set appropriately:
- For public repositories: Set to **Public**
- For private repositories: Configure access as needed

## Example Release Commands

```bash
# Patch release (1.0.0 → 1.0.1)
git tag v1.0.1
git push origin v1.0.1

# Minor release (1.0.1 → 1.1.0)
git tag v1.1.0
git push origin v1.1.0

# Major release (1.1.0 → 2.0.0)
git tag v2.0.0
git push origin v2.0.0
```

## Troubleshooting

### Permission Denied on Docker Push

If you see permission errors when pushing to ghcr.io:
1. Check that workflow permissions are set to "Read and write"
2. Ensure you're using GITHUB_TOKEN (already configured in the workflow)
3. Verify the package visibility settings in your GitHub account

### Build Failures

Check the Actions tab in your repository to see detailed logs:
- Go to **Actions** → Select the failed workflow run
- Review the logs for specific error messages

### Docker Image Not Appearing

1. Wait a few minutes after the workflow completes
2. Check the **Packages** section of your repository
3. Verify the package visibility is set to Public (for public repos)
4. Pull the image explicitly: `docker pull ghcr.io/{owner}/traefik-config-middleware:latest`

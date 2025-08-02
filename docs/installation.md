# Installation Guide

reviewtask provides multiple installation methods to suit different environments and preferences.

## Quick Install (Recommended)

The easiest way to install reviewtask is using our one-liner installation scripts:

=== "Unix/Linux/macOS"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash
    ```

=== "Windows (PowerShell)"

    ```powershell
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex
    ```

## Installation Locations

By default, reviewtask is installed to user directories (no sudo required):

- **Unix/Linux/macOS**: `~/.local/bin`
- **Windows**: `%USERPROFILE%\bin` (e.g., `C:\Users\username\bin`)

## PATH Configuration

After installation, you may need to add the installation directory to your PATH:

=== "Bash"

    ```bash
    # Add to ~/.bashrc
    export PATH="$HOME/.local/bin:$PATH"
    
    # Reload configuration
    source ~/.bashrc
    ```

=== "Zsh"

    ```bash
    # Add to ~/.zshrc
    export PATH="$HOME/.local/bin:$PATH"
    
    # Reload configuration
    source ~/.zshrc
    ```

=== "Fish"

    ```fish
    # Add to ~/.config/fish/config.fish
    set -gx PATH $HOME/.local/bin $PATH
    
    # Reload configuration
    source ~/.config/fish/config.fish
    ```

=== "Windows"

    The installer will provide instructions for adding the installation directory to your PATH.

## Installation Options

### Specific Version

=== "Unix/Linux/macOS"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --version v1.2.3
    ```

=== "Windows"

    ```powershell
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-Version", "v1.2.3"
    ```

### Custom Installation Directory

=== "Unix/Linux/macOS"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --bin-dir ~/bin
    ```

=== "Windows"

    ```powershell
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-BinDir", "C:\tools"
    ```

### System-wide Installation

For system-wide installation (requires admin privileges):

=== "Unix/Linux/macOS"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | sudo bash -s -- --bin-dir /usr/local/bin
    ```

=== "Windows"

    Run PowerShell as Administrator:
    ```powershell
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-BinDir", "C:\Program Files\reviewtask"
    ```

### Force Overwrite

To overwrite an existing installation:

=== "Unix/Linux/macOS"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --force
    ```

=== "Windows"

    ```powershell
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-Force"
    ```

## Alternative Installation Methods

### Manual Download

1. Download the latest release for your platform from the [releases page](https://github.com/biwakonbu/reviewtask/releases/latest)
2. Extract the archive
3. Move the binary to a directory in your PATH

### Go Install

If you have Go installed:

```bash
go install github.com/biwakonbu/reviewtask@latest
```

### Package Managers

Package manager support is planned for future releases.

## Verify Installation

After installation, verify that reviewtask is working:

```bash
reviewtask version
```

You should see output similar to:
```
reviewtask version 1.7.1
Commit: abc1234
Built: 2024-01-01T10:00:00Z
Go version: go1.21.0
OS/Arch: linux/amd64
```

## Prerequisites

### AI Provider CLI

reviewtask requires an AI provider CLI for analysis. Claude Code is recommended:

1. Install Claude Code following the [official instructions](https://docs.anthropic.com/en/docs/claude-code)
2. Verify installation: `claude --version`

### GitHub Access

You'll need:
- A GitHub account
- Access to repositories you want to analyze
- A GitHub personal access token (set up during authentication)

## Troubleshooting Installation

### Permission Denied

If you get permission errors:

1. Try installing to a user directory (default behavior)
2. Use a custom directory you have write access to
3. For system installation, ensure you have admin privileges

### Binary Not Found

If the `reviewtask` command isn't found after installation:

1. Check that the installation directory is in your PATH
2. Restart your terminal or reload your shell configuration
3. Verify the binary was installed correctly

### Checksum Verification Failed

If checksum verification fails:

1. Check your internet connection
2. Retry the installation
3. Report the issue if it persists

### Unsupported Platform

Currently supported platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

For other platforms, consider building from source.

## Updating reviewtask

### Using Built-in Updater

```bash
# Check for updates
reviewtask version --check

# Update to latest version
reviewtask version latest

# Update to specific version
reviewtask version v1.2.3
```

### Manual Update

Re-run the installation script to get the latest version:

```bash
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --force
```

## Uninstallation

To remove reviewtask:

1. Delete the binary from your installation directory
2. Remove the PATH configuration from your shell profile
3. Optionally remove the `.pr-review/` directories from your projects

=== "Unix/Linux/macOS"

    ```bash
    # Remove binary
    rm ~/.local/bin/reviewtask
    
    # Remove PATH configuration
    # Edit ~/.bashrc, ~/.zshrc, or ~/.config/fish/config.fish
    ```

=== "Windows"

    ```powershell
    # Remove binary
    Remove-Item "$env:USERPROFILE\bin\reviewtask.exe"
    
    # Remove from PATH through System Properties or PowerShell profile
    ```

## Security Considerations

The installation scripts:
- Download from GitHub releases using HTTPS
- Verify SHA256 checksums for binary integrity
- Install to user directories by default to minimize privilege requirements
- Follow security best practices for shell script distribution

For enhanced security, you can download and verify the installation scripts manually before running them.

## Getting Help

If you encounter issues during installation:

1. Check this troubleshooting section
2. Review the [detailed installation documentation](https://github.com/biwakonbu/reviewtask/blob/main/docs/INSTALLATION.md)
3. Open an issue on the [GitHub repository](https://github.com/biwakonbu/reviewtask/issues)

Next: After installation, proceed to the [Quick Start Guide](quick-start.md) to set up reviewtask in your repository.
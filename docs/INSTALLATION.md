# Installation System Documentation

This document provides comprehensive documentation for the reviewtask installation system, including the curl-based installation scripts and their implementation details.

## Overview

The reviewtask installation system provides cross-platform, one-liner installation scripts that automatically detect the user's platform, download the appropriate binary, verify its integrity, and install it in the correct location.

## Installation Scripts

### 1. Unix/Linux/macOS Script (`install.sh`)

**Location**: `scripts/install/install.sh`  
**Purpose**: Provides installation for Unix-like systems including Linux and macOS  
**Language**: Bash shell script  
**Compatibility**: bash 3.0+, compatible with most Unix shells

#### Key Features

- **Platform Auto-Detection**: Automatically detects OS (Linux/Darwin) and architecture (amd64/arm64)
- **Version Management**: Supports latest, specific versions, and pre-release versions
- **Security**: SHA256 checksum verification for downloaded binaries
- **Flexibility**: Configurable installation directory and force overwrite options
- **Error Handling**: Comprehensive error checking with rollback capabilities
- **Network Resilience**: Falls back between curl and wget for downloads
- **Shell Detection**: Automatically detects user's shell (bash, zsh, fish) for PATH configuration
- **Archive Support**: Handles tar.gz and zip archives for binary distribution

#### Usage Examples

```bash
# Basic installation (secure two-step process)
curl -fsSL -o install.sh https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh.sha256 | sha256sum -c
bash install.sh

# Install specific version
curl -fsSL -o install.sh https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh.sha256 | sha256sum -c
bash install.sh --version v1.2.3

# Install to custom directory
curl -fsSL -o install.sh https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh.sha256 | sha256sum -c
bash install.sh --bin-dir ~/bin

# Force overwrite existing installation
curl -fsSL -o install.sh https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh.sha256 | sha256sum -c
bash install.sh --force

# Include pre-release versions
curl -fsSL -o install.sh https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh.sha256 | sha256sum -c
bash install.sh --prerelease
```

#### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--version VERSION` | Install specific version | `latest` |
| `--bin-dir DIR` | Installation directory | `~/.local/bin` |
| `--force` | Overwrite existing installation | `false` |
| `--prerelease` | Include pre-release versions | `false` |
| `--help` | Show usage information | - |

### 2. Windows PowerShell Script (`install.ps1`)

**Location**: `scripts/install/install.ps1`  
**Purpose**: Provides installation for Windows systems using PowerShell  
**Language**: PowerShell  
**Compatibility**: PowerShell 3.0+ (Windows PowerShell and PowerShell Core)

#### Key Features

- **Platform Auto-Detection**: Automatically detects Windows architecture (amd64/arm64)
- **Native PowerShell**: Uses PowerShell-native cmdlets and functions
- **Security**: SHA256 hash verification using Get-FileHash
- **User-Friendly**: Colored output and comprehensive error messages
- **PATH Integration**: Automatic PATH detection and modification guidance

#### Usage Examples

```powershell
# Basic installation (secure two-step process)
iwr -useb -o install.ps1 https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1
$expectedHash = (iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1.sha256).Content
if ((Get-FileHash install.ps1).Hash -eq $expectedHash) { .\install.ps1 } else { Write-Error "Checksum verification failed" }

# Install specific version
iwr -useb -o install.ps1 https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1
$expectedHash = (iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1.sha256).Content
if ((Get-FileHash install.ps1).Hash -eq $expectedHash) { .\install.ps1 -Version "v1.2.3" } else { Write-Error "Checksum verification failed" }

# Install to custom directory
iwr -useb -o install.ps1 https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1
$expectedHash = (iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1.sha256).Content
if ((Get-FileHash install.ps1).Hash -eq $expectedHash) { .\install.ps1 -BinDir "C:\tools" } else { Write-Error "Checksum verification failed" }

# Force overwrite existing installation
iwr -useb -o install.ps1 https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1
$expectedHash = (iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1.sha256).Content
if ((Get-FileHash install.ps1).Hash -eq $expectedHash) { .\install.ps1 -Force } else { Write-Error "Checksum verification failed" }
```

#### Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-Version` | Install specific version | `latest` |
| `-BinDir` | Installation directory | `$env:USERPROFILE\bin` |
| `-Force` | Overwrite existing installation | `false` |
| `-Prerelease` | Include pre-release versions | `false` |
| `-Help` | Show usage information | - |

## Architecture and Implementation

### Platform Detection Logic

#### Unix/Linux/macOS (`install.sh`)

```bash
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*)
            print_error "Windows detected. Please use install.ps1"
            exit 1
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}
```

#### Windows PowerShell (`install.ps1`)

```powershell
function Get-Platform {
    $arch = $env:PROCESSOR_ARCHITECTURE
    
    switch ($arch) {
        "AMD64" { return "windows_amd64" }
        "ARM64" { return "windows_arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}
```

### Version Resolution

Both scripts support multiple version resolution strategies:

1. **Latest Release**: Default behavior, fetches latest stable release
2. **Specific Version**: User-specified version with validation
3. **Pre-release**: Includes alpha, beta, and release candidate versions

#### GitHub API Integration

```bash
# Unix/Linux/macOS
get_latest_version() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases"
    
    if [[ "$PRERELEASE" == "true" ]]; then
        api_url="${api_url}"
    else
        api_url="${api_url}/latest"
    fi

    curl -s "$api_url" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/'
}
```

```powershell
# Windows PowerShell
function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/$GitHubRepo/releases"
    
    if ($Prerelease) {
        $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
        return $response[0].tag_name
    } else {
        $apiUrl = "$apiUrl/latest"
        $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
        return $response.tag_name
    }
}
```

### Security and Verification

#### Checksum Verification Process

1. **Download Binary**: Fetch the appropriate binary for detected platform
2. **Download Checksums**: Fetch checksums.txt from the same release
3. **Extract Expected Hash**: Parse checksum file for binary filename
4. **Calculate Actual Hash**: Compute SHA256 of downloaded binary
5. **Compare and Verify**: Ensure hashes match exactly

#### Unix/Linux/macOS Implementation

```bash
download_with_verification() {
    local url="$1"
    local output_file="$2"
    local checksum_url="$3"
    
    # Download the binary
    curl -fsSL "$url" -o "$output_file"
    
    # Verify checksum if available
    if [[ -n "$checksum_url" ]] && command -v sha256sum >/dev/null 2>&1; then
        local expected_checksum
        expected_checksum=$(curl -fsSL "$checksum_url" | grep "$(basename "$output_file")" | awk '{print $1}')
        
        if [[ -n "$expected_checksum" ]]; then
            local actual_checksum
            actual_checksum=$(sha256sum "$output_file" | awk '{print $1}')
            
            if [[ "$actual_checksum" != "$expected_checksum" ]]; then
                print_error "Checksum verification failed"
                rm -f "$output_file"
                exit 1
            fi
        fi
    fi
}
```

#### Windows PowerShell Implementation

```powershell
function Get-FileWithVerification {
    param(
        [string]$Url,
        [string]$OutputPath,
        [string]$ChecksumUrl
    )
    
    # Download binary
    Invoke-WebRequest -Uri $Url -OutFile $OutputPath -UseBasicParsing
    
    # Verify checksum if available
    if ($ChecksumUrl -and (Get-Command Get-FileHash -ErrorAction SilentlyContinue)) {
        $checksumContent = Invoke-WebRequest -Uri $ChecksumUrl -UseBasicParsing | Select-Object -ExpandProperty Content
        $fileName = Split-Path $OutputPath -Leaf
        $expectedChecksum = ($checksumContent -split "`n" | Where-Object { $_ -match $fileName } | ForEach-Object { ($_ -split '\s+')[0] })
        
        if ($expectedChecksum) {
            $actualChecksum = (Get-FileHash -Path $OutputPath -Algorithm SHA256).Hash.ToLower()
            
            if ($actualChecksum -ne $expectedChecksum.ToLower()) {
                Write-Error "Checksum verification failed"
                Remove-Item $OutputPath -ErrorAction SilentlyContinue
                exit 1
            }
        }
    }
}
```

### Error Handling and Recovery

#### Comprehensive Error Scenarios

1. **Network Connectivity**: Graceful handling of download failures
2. **Permission Issues**: Clear guidance for installation directory problems
3. **Existing Installation**: Detection and handling of existing binaries
4. **Invalid Versions**: Validation of version format and availability
5. **Checksum Failures**: Security validation with rollback capabilities

#### Rollback Strategy

Both scripts implement automatic rollback on failure:

1. **Temporary Files**: All downloads use temporary locations initially
2. **Atomic Operations**: Final installation is an atomic move operation
3. **Cleanup on Error**: Temporary files are cleaned up on any failure
4. **State Preservation**: Existing installations are not modified until success

### Installation Directory Strategy

#### Default Locations

| Platform | Default Directory | Reasoning |
|----------|-------------------|-----------|
| Unix/Linux | `~/.local/bin` | User-specific directory, no sudo required |
| macOS | `~/.local/bin` | User-specific directory, no sudo required |
| Windows | `$env:USERPROFILE\bin` | User-specific to avoid admin requirements |

#### Permission Handling

- **Unix/Linux/macOS**: Defaults to user directory (`~/.local/bin`) to avoid sudo requirements
- **Windows**: Defaults to user directory to avoid UAC requirements
- **Both**: Provides shell-specific guidance for PATH configuration

#### Shell-Specific PATH Configuration

The installation script automatically detects the user's shell and provides appropriate PATH configuration instructions:

**Bash Configuration:**
```bash
# Add to ~/.bashrc
export PATH="$HOME/.local/bin:$PATH"
source ~/.bashrc
```

**Zsh Configuration:**
```bash
# Add to ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
source ~/.zshrc
```

**Fish Configuration:**
```fish
# Add to ~/.config/fish/config.fish
set -gx PATH $HOME/.local/bin $PATH
source ~/.config/fish/config.fish
```

### Supported Binary Formats

The installation system supports archived binaries with the following naming convention:

```
reviewtask-<version>-<platform>-<architecture>.tar.gz  # Unix/Linux/macOS
reviewtask-<version>-<platform>-<architecture>.zip     # Windows
```

#### Platform Identifiers

| Platform | Identifier | Archive Name Example |
|----------|------------|-------------------|
| Linux x86_64 | `linux-amd64` | `reviewtask-v0.1.0-linux-amd64.tar.gz` |
| Linux ARM64 | `linux-arm64` | `reviewtask-v0.1.0-linux-arm64.tar.gz` |
| macOS x86_64 | `darwin-amd64` | `reviewtask-v0.1.0-darwin-amd64.tar.gz` |
| macOS ARM64 | `darwin-arm64` | `reviewtask-v0.1.0-darwin-arm64.tar.gz` |
| Windows x86_64 | `windows-amd64` | `reviewtask-v0.1.0-windows-amd64.zip` |
| Windows ARM64 | `windows-arm64` | `reviewtask-v0.1.0-windows-arm64.zip` |

## Testing Framework

### Test Suites

#### Unix/Linux/macOS Tests (`test_install.sh`)

**Coverage Areas:**
- Help display functionality
- Platform detection accuracy
- Version format validation
- Directory creation and permissions
- Existing installation detection
- Force overwrite functionality
- Argument parsing correctness
- Network functionality (with mocking)
- Error handling scenarios
- Script permissions validation

#### Windows PowerShell Tests (`test_install.ps1`)

**Coverage Areas:**
- Script syntax validation
- Platform detection for Windows
- Version format validation
- Directory creation and permissions
- Existing installation detection
- Force overwrite functionality
- Parameter handling
- Help display functionality
- Error handling scenarios
- Utility function validation

### Running Tests

#### Unix/Linux/macOS

```bash
# Run all tests
./test_install.sh

# Run with verbose output
./test_install.sh --verbose
```

#### Windows PowerShell

```powershell
# Run all tests
.\test_install.ps1

# Run with verbose output
.\test_install.ps1 -Verbose
```

### Test Environment

Tests run in isolated environments:

- **Temporary Directories**: Each test uses clean temporary directories
- **Mock Functions**: External dependencies are mocked when possible
- **Network Tests**: Real network tests are conditional on connectivity
- **Cross-Platform**: Tests account for platform-specific behaviors

## Deployment and Distribution

### GitHub Integration

The installation scripts are designed to work seamlessly with GitHub releases:

1. **Raw File Access**: Scripts are accessed via GitHub's raw file API
2. **Release Assets**: Binaries are downloaded from GitHub release assets
3. **API Integration**: Release information is fetched via GitHub API
4. **Checksum Files**: Checksums are distributed as release assets

### URL Structure

```
# Script URLs
https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh
https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1

# Archive URLs  
https://github.com/biwakonbu/reviewtask/releases/download/v1.2.3/reviewtask-v1.2.3-linux-amd64.tar.gz
https://github.com/biwakonbu/reviewtask/releases/download/v1.2.3/reviewtask-v1.2.3-windows-amd64.zip

# Checksum URL
https://github.com/biwakonbu/reviewtask/releases/download/v1.2.3/SHA256SUMS
```

### Content Delivery Network (CDN)

GitHub's raw file delivery provides:

- **Global Distribution**: Worldwide CDN for fast downloads
- **High Availability**: Redundant infrastructure
- **SSL/TLS**: Secure delivery by default
- **Caching**: Appropriate cache headers for performance

## Best Practices and Recommendations

### For Users

1. **Always Use HTTPS**: The provided URLs use HTTPS by default
2. **Verify Installation**: Run `reviewtask version` after installation
3. **Check PATH**: Ensure installation directory is in PATH
4. **Regular Updates**: Use the same script to update to newer versions

### For Developers

1. **Test Locally**: Always test installation scripts before release
2. **Maintain Checksums**: Generate and distribute checksum files
3. **Version Validation**: Ensure version tags follow semantic versioning
4. **Documentation**: Keep installation documentation current

### Security Considerations

1. **Checksum Verification**: Always verify binary integrity
2. **HTTPS Only**: Never use HTTP for downloads
3. **Input Validation**: Validate all user inputs and parameters
4. **Minimal Privileges**: Default to user directories when possible
5. **Error Handling**: Fail securely and provide clear error messages

## Troubleshooting

### Common Issues

#### Permission Denied

```bash
# Solution: Use custom directory
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --bin-dir ~/bin
```

#### Binary Not in PATH

```bash
# Add to shell profile
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### Checksum Verification Failed

```bash
# Retry installation (may be temporary network issue)
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --force
```

#### Unsupported Platform

Check platform support and consider manual installation from releases page.

### Debug Mode

Both scripts provide verbose output for debugging:

```bash
# Unix/Linux/macOS - Enable debug output
bash -x <(curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh)

# Windows PowerShell - Use verbose preference
$VerbosePreference = "Continue"
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex
```

## Future Enhancements

### Planned Features

1. **Package Manager Integration**: Homebrew, Chocolatey, apt/yum repositories
2. **Auto-Update Mechanism**: Built-in update checking and installation
3. **Configuration Profiles**: Pre-configured installation profiles
4. **Offline Installation**: Support for air-gapped environments
5. **Digital Signatures**: Code signing for enhanced security

### Extensibility

The installation system is designed to be extensible:

- **Additional Platforms**: Easy to add new platform support
- **Custom Sources**: Support for custom binary repositories
- **Plugin Architecture**: Modular verification and installation plugins
- **Configuration Management**: Enhanced configuration file support

---

This documentation provides comprehensive coverage of the reviewtask installation system. For the latest updates and additional information, refer to the project's GitHub repository and release notes.
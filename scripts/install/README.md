# Installation Scripts

This directory contains the official installation scripts for reviewtask across different platforms.

## Files

- **`install.sh`** - Unix/Linux/macOS installation script
- **`install.ps1`** - Windows PowerShell installation script
- **`test_install.sh`** - Test suite for Unix/Linux/macOS script
- **`test_install.ps1`** - Test suite for Windows PowerShell script

## Quick Installation

### Unix/Linux/macOS
```bash
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash
```

### Windows PowerShell
```powershell
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex
```

## Features

- **Cross-Platform Support**: Works on Linux, macOS, and Windows
- **Platform Auto-Detection**: Automatically detects OS and architecture
- **Security**: SHA256 checksum verification for all downloads
- **Flexible Installation**: Configurable installation directory
- **Version Management**: Support for specific versions and pre-releases
- **Error Handling**: Comprehensive error checking with rollback capabilities

## Testing

Run the test suites to verify script functionality:

### Unix/Linux/macOS
```bash
./test_install.sh
```

### Windows PowerShell
```powershell
.\test_install.ps1
```

## Documentation

For comprehensive documentation, see [docs/INSTALLATION.md](../../docs/INSTALLATION.md).

## Command Line Options

### Unix/Linux/macOS (`install.sh`)

| Option | Description | Default |
|--------|-------------|---------|
| `--version VERSION` | Install specific version | `latest` |
| `--bin-dir DIR` | Installation directory | `/usr/local/bin` |
| `--force` | Overwrite existing installation | `false` |
| `--prerelease` | Include pre-release versions | `false` |
| `--help` | Show usage information | - |

### Windows PowerShell (`install.ps1`)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-Version` | Install specific version | `latest` |
| `-BinDir` | Installation directory | `$env:USERPROFILE\bin` |
| `-Force` | Overwrite existing installation | `false` |
| `-Prerelease` | Include pre-release versions | `false` |
| `-Help` | Show usage information | - |

## Examples

### Install Latest Version
```bash
# Unix/Linux/macOS
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash

# Windows
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex
```

### Install Specific Version
```bash
# Unix/Linux/macOS
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --version v1.2.3

# Windows
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-Version", "v1.2.3"
```

### Custom Installation Directory
```bash
# Unix/Linux/macOS
curl -fsSL https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.sh | bash -s -- --bin-dir ~/bin

# Windows
iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1 | iex -ArgumentList "-BinDir", "C:\tools"
```

## Security

All installation scripts include the following security features:

1. **HTTPS Only**: All downloads use secure HTTPS connections
2. **Checksum Verification**: SHA256 hash verification for binary integrity
3. **Input Validation**: Comprehensive validation of all user inputs
4. **Secure Defaults**: Safe default values and minimal privilege requirements
5. **Error Handling**: Graceful failure with secure cleanup

## Support

For issues with installation scripts:

1. Check the [troubleshooting guide](../../docs/INSTALLATION.md#troubleshooting)
2. Run with debug output for detailed information
3. Open an issue on GitHub with system details and error messages
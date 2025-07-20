# reviewtask PowerShell Installation Script
# Automatically detects platform and installs the appropriate binary for Windows

param(
    [string]$Version = "latest",
    [string]$BinDir = "$env:USERPROFILE\bin",
    [switch]$Force,
    [switch]$Prerelease,
    [switch]$Help
)

# Configuration
$GitHubRepo = "biwakonbu/reviewtask"
$BinaryName = "reviewtask.exe"
$ErrorActionPreference = "Stop"

# Function to write colored output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Show usage information
function Show-Usage {
    @"
reviewtask PowerShell Installation Script

USAGE:
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.ps1 | iex
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.ps1 | iex -ArgumentList [OPTIONS]

OPTIONS:
    -Version VERSION     Install specific version (default: latest)
    -BinDir DIR         Installation directory (default: %USERPROFILE%\bin)
    -Force              Overwrite existing installation
    -Prerelease         Include pre-release versions
    -Help               Show this help message

EXAMPLES:
    # Install latest version to default location
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.ps1 | iex

    # Install specific version
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.ps1 | iex -ArgumentList "-Version", "v1.2.3"

    # Install to custom directory
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.ps1 | iex -ArgumentList "-BinDir", "C:\tools"

    # Force overwrite existing installation
    iwr -useb https://raw.githubusercontent.com/biwakonbu/reviewtask/main/install.ps1 | iex -ArgumentList "-Force"
"@
}

# Detect platform architecture
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

# Get latest release version from GitHub API
function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/$GitHubRepo/releases"
    
    if ($Prerelease) {
        $apiUrl = "$apiUrl"
    } else {
        $apiUrl = "$apiUrl/latest"
    }
    
    try {
        if ($Prerelease) {
            $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
            return $response[0].tag_name
        } else {
            $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
            return $response.tag_name
        }
    }
    catch {
        Write-Error "Failed to get latest version: $_"
        exit 1
    }
}

# Validate version format
function Test-VersionFormat {
    param([string]$Version)
    
    if ($Version -eq "latest") {
        return $true
    }
    
    # Check if version starts with 'v' followed by semantic version
    if ($Version -notmatch '^v\d+\.\d+\.\d+([+-][a-zA-Z0-9.-]*)?$') {
        Write-Error "Invalid version format: $Version"
        Write-Info "Version should be in format: v1.2.3 or v1.2.3-beta.1"
        exit 1
    }
    
    return $true
}

# Check if binary already exists
function Test-ExistingInstallation {
    $binaryPath = Join-Path $BinDir $BinaryName
    
    if ((Test-Path $binaryPath) -and -not $Force) {
        Write-Warning "reviewtask is already installed at $binaryPath"
        
        try {
            $currentVersion = & $binaryPath version 2>$null | Select-Object -First 1 | ForEach-Object { ($_ -split '\s+')[2] }
            Write-Info "Current version: $currentVersion"
        }
        catch {
            Write-Info "Current version: unknown"
        }
        
        Write-Info "Use -Force to overwrite the existing installation"
        exit 1
    }
}

# Create installation directory if it doesn't exist
function New-InstallDirectory {
    if (-not (Test-Path $BinDir)) {
        Write-Info "Creating installation directory: $BinDir"
        try {
            New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
        }
        catch {
            Write-Error "Failed to create directory $BinDir: $_"
            exit 1
        }
    }
    
    # Test if directory is writable
    $testFile = Join-Path $BinDir "test-write-access.tmp"
    try {
        [System.IO.File]::WriteAllText($testFile, "test")
        Remove-Item $testFile -ErrorAction SilentlyContinue
    }
    catch {
        Write-Error "Directory $BinDir is not writable: $_"
        Write-Info "You may need to run as Administrator or choose a different directory"
        exit 1
    }
}

# Download file with checksum verification
function Get-FileWithVerification {
    param(
        [string]$Url,
        [string]$OutputPath,
        [string]$ChecksumUrl
    )
    
    Write-Info "Downloading $Url"
    
    try {
        Invoke-WebRequest -Uri $Url -OutFile $OutputPath -UseBasicParsing
    }
    catch {
        Write-Error "Failed to download $Url: $_"
        exit 1
    }
    
    # Verify checksum if available
    if ($ChecksumUrl -and (Get-Command Get-FileHash -ErrorAction SilentlyContinue)) {
        Write-Info "Verifying checksum..."
        
        try {
            $checksumContent = Invoke-WebRequest -Uri $ChecksumUrl -UseBasicParsing | Select-Object -ExpandProperty Content
            $fileName = Split-Path $OutputPath -Leaf
            $expectedChecksum = ($checksumContent -split "`n" | Where-Object { $_ -match $fileName } | ForEach-Object { ($_ -split '\s+')[0] })
            
            if ($expectedChecksum) {
                $actualChecksum = (Get-FileHash -Path $OutputPath -Algorithm SHA256).Hash.ToLower()
                
                if ($actualChecksum -ne $expectedChecksum.ToLower()) {
                    Write-Error "Checksum verification failed"
                    Write-Error "Expected: $expectedChecksum"
                    Write-Error "Actual: $actualChecksum"
                    Remove-Item $OutputPath -ErrorAction SilentlyContinue
                    exit 1
                }
                Write-Success "Checksum verification passed"
            }
            else {
                Write-Warning "Could not retrieve checksum for verification"
            }
        }
        catch {
            Write-Warning "Checksum verification failed: $_"
        }
    }
}

# Install the binary
function Install-Binary {
    param(
        [string]$Platform,
        [string]$Version
    )
    
    # Resolve latest version if needed
    if ($Version -eq "latest") {
        Write-Info "Resolving latest version..."
        $Version = Get-LatestVersion
        if (-not $Version) {
            Write-Error "Failed to determine latest version"
            exit 1
        }
        Write-Info "Latest version: $Version"
    }
    
    Test-VersionFormat -Version $Version
    
    # Construct download URLs
    $binaryFilename = "reviewtask_$Platform.exe"
    $downloadUrl = "https://github.com/$GitHubRepo/releases/download/$Version/$binaryFilename"
    $checksumUrl = "https://github.com/$GitHubRepo/releases/download/$Version/checksums.txt"
    
    # Create temporary file
    $tempDir = [System.IO.Path]::GetTempPath()
    $tempBinary = Join-Path $tempDir $binaryFilename
    
    try {
        # Download and verify the binary
        Get-FileWithVerification -Url $downloadUrl -OutputPath $tempBinary -ChecksumUrl $checksumUrl
        
        # Move to final location
        $finalPath = Join-Path $BinDir $BinaryName
        Write-Info "Installing to $finalPath"
        
        Move-Item $tempBinary $finalPath -Force
        
        Write-Success "Successfully installed reviewtask $Version to $finalPath"
    }
    finally {
        # Cleanup
        if (Test-Path $tempBinary) {
            Remove-Item $tempBinary -ErrorAction SilentlyContinue
        }
    }
}

# Verify installation
function Test-Installation {
    $binaryPath = Join-Path $BinDir $BinaryName
    
    Write-Info "Verifying installation..."
    
    # Check if binary exists
    if (-not (Test-Path $binaryPath)) {
        Write-Error "Binary not found: $binaryPath"
        exit 1
    }
    
    # Check if binary works
    try {
        $null = & $binaryPath version 2>$null
    }
    catch {
        Write-Error "Binary verification failed: $binaryPath version"
        exit 1
    }
    
    try {
        $installedVersion = & $binaryPath version 2>$null | Select-Object -First 1 | ForEach-Object { ($_ -split '\s+')[2] }
        Write-Success "Installation verified successfully"
        Write-Info "Installed version: $installedVersion"
    }
    catch {
        Write-Success "Installation verified successfully"
        Write-Info "Installed version: unknown"
    }
    
    # Check if binary is in PATH
    $pathDirs = $env:PATH -split ';'
    $isInPath = $pathDirs -contains $BinDir
    
    if (-not $isInPath) {
        Write-Warning "$BinDir is not in your PATH"
        Write-Info "Add $BinDir to your PATH environment variable to use reviewtask from anywhere."
        Write-Info "Or run reviewtask with full path: $binaryPath"
        Write-Info ""
        Write-Info "To add to PATH permanently, run:"
        Write-Info '[Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";' + $BinDir + '", "User")'
    }
    else {
        Write-Success "reviewtask is available in your PATH"
        Write-Info "You can now run: reviewtask --help"
    }
}

# Main installation function
function Install-Reviewtask {
    Write-Info "reviewtask PowerShell Installation Script"
    Write-Info "Repository: https://github.com/$GitHubRepo"
    
    # Show help if requested
    if ($Help) {
        Show-Usage
        return
    }
    
    # Detect platform
    $platform = Get-Platform
    Write-Info "Detected platform: $platform"
    
    # Show configuration
    Write-Info "Configuration:"
    Write-Info "  Version: $Version"
    Write-Info "  Install directory: $BinDir"
    Write-Info "  Force overwrite: $Force"
    Write-Info "  Include prereleases: $Prerelease"
    
    # Check existing installation
    Test-ExistingInstallation
    
    # Create installation directory
    New-InstallDirectory
    
    # Install binary
    Install-Binary -Platform $platform -Version $Version
    
    # Verify installation
    Test-Installation
    
    Write-Success "Installation completed successfully!"
}

# Run main function
try {
    Install-Reviewtask
}
catch {
    Write-Error "Installation failed: $_"
    exit 1
}
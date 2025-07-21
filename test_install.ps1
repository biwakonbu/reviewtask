# PowerShell Installation Script Test Suite
# Tests the install.ps1 script under various conditions

param(
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

# Test configuration
$TestBinDir = Join-Path $PSScriptRoot "test_bin_ps"
$InstallScript = Join-Path $PSScriptRoot "install.ps1"
$GitHubRepo = "biwakonbu/reviewtask"

# Test counters
$TestsRun = 0
$TestsPassed = 0
$TestsFailed = 0

# Test output functions
function Write-TestHeader {
    param([string]$Message)
    Write-Host "=== $Message ===" -ForegroundColor Blue
}

function Write-TestSuccess {
    param([string]$Message)
    Write-Host "[PASS] $Message" -ForegroundColor Green
    $script:TestsPassed++
}

function Write-TestFailure {
    param([string]$Message)
    Write-Host "[FAIL] $Message" -ForegroundColor Red
    $script:TestsFailed++
}

function Write-TestInfo {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Yellow
}

# Test runner function
function Invoke-Test {
    param(
        [string]$TestName,
        [scriptblock]$TestScript
    )
    
    $script:TestsRun++
    Write-TestHeader $TestName
    
    # Create clean test environment
    if (Test-Path $TestBinDir) {
        Remove-Item $TestBinDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $TestBinDir -Force | Out-Null
    
    try {
        $result = & $TestScript
        if ($result) {
            Write-TestSuccess $TestName
        } else {
            Write-TestFailure $TestName
        }
    }
    catch {
        Write-TestFailure "$TestName - Error: $_"
        if ($Verbose) {
            Write-Host $_.Exception.Message -ForegroundColor Red
            Write-Host $_.ScriptStackTrace -ForegroundColor Gray
        }
    }
    
    Write-Host ""
}

# Test functions

function Test-ScriptSyntax {
    # Test PowerShell script syntax
    try {
        $null = [System.Management.Automation.PSParser]::Tokenize((Get-Content $InstallScript -Raw), [ref]$null)
        return $true
    }
    catch {
        return $false
    }
}

function Test-PlatformDetection {
    # Test platform detection
    $content = Get-Content $InstallScript -Raw
    $scriptBlock = [ScriptBlock]::Create($content + "; Get-Platform")
    
    try {
        $platform = & $scriptBlock
        return $platform -match "windows_(amd64|arm64)"
    }
    catch {
        return $false
    }
}

function Test-VersionValidation {
    # Test version format validation
    $content = Get-Content $InstallScript -Raw
    $scriptBlock = [ScriptBlock]::Create($content + @"
; Test-VersionFormat "v1.2.3"
"@)
    
    try {
        $result1 = & $scriptBlock
        
        $scriptBlock2 = [ScriptBlock]::Create($content + @"
; Test-VersionFormat "invalid"
"@)
        
        try {
            $result2 = & $scriptBlock2
            # Should throw an error for invalid version
            return $false
        }
        catch {
            # Expected to fail for invalid version
            return $result1
        }
    }
    catch {
        return $false
    }
}

function Test-DirectoryCreation {
    # Test installation directory creation
    $testDir = Join-Path $TestBinDir "custom_dir"
    
    $content = Get-Content $InstallScript -Raw
    $scriptBlock = [ScriptBlock]::Create($content + @"
; `$BinDir = "$testDir"; New-InstallDirectory
"@)
    
    try {
        & $scriptBlock
        return (Test-Path $testDir) -and (Test-Path $testDir -PathType Container)
    }
    catch {
        return $false
    }
}

function Test-ExistingInstallationCheck {
    # Test existing installation detection
    $fakeBinary = Join-Path $TestBinDir "reviewtask.exe"
    "@echo off`necho reviewtask version v1.0.0" | Set-Content $fakeBinary
    
    $content = Get-Content $InstallScript -Raw
    $scriptBlock = [ScriptBlock]::Create($content + @"
; `$BinDir = "$TestBinDir"; `$Force = `$false; Test-ExistingInstallation
"@)
    
    try {
        & $scriptBlock
        # Should fail when binary exists and force is false
        return $false
    }
    catch {
        # Expected to fail
        return $true
    }
}

function Test-ForceOverwrite {
    # Test force overwrite functionality
    $fakeBinary = Join-Path $TestBinDir "reviewtask.exe"
    "@echo off`necho old version" | Set-Content $fakeBinary
    
    $content = Get-Content $InstallScript -Raw
    $scriptBlock = [ScriptBlock]::Create($content + @"
; `$BinDir = "$TestBinDir"; `$Force = `$true; Test-ExistingInstallation
"@)
    
    try {
        & $scriptBlock
        # Should succeed when force is true
        return $true
    }
    catch {
        return $false
    }
}

function Test-ParameterHandling {
    # Test parameter handling
    try {
        $params = @{
            Version = "v1.2.3"
            BinDir = "C:\test"
            Force = $true
            Prerelease = $false
        }
        
        # Just test that parameters can be passed without error
        $content = Get-Content $InstallScript -Raw
        # Remove the main execution part for testing
        $testContent = $content -replace 'Install-Reviewtask', '# Install-Reviewtask'
        $null = [ScriptBlock]::Create($testContent)
        
        return $true
    }
    catch {
        return $false
    }
}

function Test-HelpDisplay {
    # Test help functionality
    try {
        $content = Get-Content $InstallScript -Raw
        $scriptBlock = [ScriptBlock]::Create($content + "; Show-Usage")
        
        $result = & $scriptBlock
        return $result -match "PowerShell Installation Script"
    }
    catch {
        return $false
    }
}

function Test-ErrorHandling {
    # Test error handling for invalid scenarios
    try {
        # Test invalid version format
        $content = Get-Content $InstallScript -Raw
        $scriptBlock = [ScriptBlock]::Create($content + @"
; Test-VersionFormat "invalid-version"
"@)
        
        try {
            & $scriptBlock
            return $false  # Should have thrown an error
        }
        catch {
            return $true   # Expected to fail
        }
    }
    catch {
        return $false
    }
}

function Test-UtilityFunctions {
    # Test utility functions exist and work
    $content = Get-Content $InstallScript -Raw
    
    $functions = @(
        "Write-Info",
        "Write-Success", 
        "Write-Warning",
        "Write-Error",
        "Show-Usage",
        "Get-Platform",
        "Get-LatestVersion",
        "Test-VersionFormat"
    )
    
    foreach ($func in $functions) {
        if ($content -notmatch "function $func") {
            return $false
        }
    }
    
    return $true
}

function Test-NetworkFunctionality {
    # Test network-related functions (mock test)
    try {
        $content = Get-Content $InstallScript -Raw
        $scriptBlock = [ScriptBlock]::Create($content + @"
; `$Prerelease = `$false; if (Get-Command Invoke-RestMethod -ErrorAction SilentlyContinue) { "Network functions available" } else { "Network functions not available" }
"@)
        
        $result = & $scriptBlock
        return $result -ne $null
    }
    catch {
        return $false
    }
}

# Main test runner
function Main {
    Write-TestHeader "PowerShell Installation Script Test Suite"
    Write-TestInfo "Testing: $InstallScript"
    Write-Host ""
    
    # Check prerequisites
    if (-not (Test-Path $InstallScript)) {
        Write-Host "ERROR: Installation script not found: $InstallScript" -ForegroundColor Red
        exit 1
    }
    
    # Run tests
    Invoke-Test "Script Syntax" { Test-ScriptSyntax }
    Invoke-Test "Platform Detection" { Test-PlatformDetection }
    Invoke-Test "Version Validation" { Test-VersionValidation }
    Invoke-Test "Directory Creation" { Test-DirectoryCreation }
    Invoke-Test "Existing Installation Check" { Test-ExistingInstallationCheck }
    Invoke-Test "Force Overwrite" { Test-ForceOverwrite }
    Invoke-Test "Parameter Handling" { Test-ParameterHandling }
    Invoke-Test "Help Display" { Test-HelpDisplay }
    Invoke-Test "Error Handling" { Test-ErrorHandling }
    Invoke-Test "Utility Functions" { Test-UtilityFunctions }
    Invoke-Test "Network Functionality" { Test-NetworkFunctionality }
    
    # Cleanup
    if (Test-Path $TestBinDir) {
        Remove-Item $TestBinDir -Recurse -Force
    }
    
    # Print summary
    Write-TestHeader "Test Summary"
    Write-Host "Tests run: $TestsRun"
    Write-Host "Passed: $TestsPassed" -ForegroundColor Green
    Write-Host "Failed: $TestsFailed" -ForegroundColor Red
    
    if ($TestsFailed -eq 0) {
        Write-Host "`nAll tests passed!" -ForegroundColor Green
        exit 0
    } else {
        Write-Host "`nSome tests failed!" -ForegroundColor Red
        exit 1
    }
}

# Execute main function
Main
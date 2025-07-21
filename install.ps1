# reviewtask PowerShell Installation Script Wrapper
# This is a lightweight wrapper that redirects to the actual installation script

param(
    [string]$Version = "latest",
    [string]$BinDir = "$env:USERPROFILE\bin",
    [switch]$Force,
    [switch]$Prerelease,
    [switch]$Help
)

Write-Host "reviewtask Installation" -ForegroundColor Blue
Write-Host "Downloading installation script..."

try {
    # Download and execute the actual installation script
    $scriptUrl = "https://raw.githubusercontent.com/biwakonbu/reviewtask/main/scripts/install/install.ps1"
    $scriptContent = Invoke-WebRequest -Uri $scriptUrl -UseBasicParsing | Select-Object -ExpandProperty Content
    
    # Create a script block and invoke it with parameters
    $scriptBlock = [ScriptBlock]::Create($scriptContent)
    
    # Build parameter hash
    $params = @{}
    if ($Version -ne "latest") { $params.Version = $Version }
    if ($BinDir -ne "$env:USERPROFILE\bin") { $params.BinDir = $BinDir }
    if ($Force) { $params.Force = $true }
    if ($Prerelease) { $params.Prerelease = $true }
    if ($Help) { $params.Help = $true }
    
    & $scriptBlock @params
}
catch {
    Write-Error "Installation failed: $_"
    exit 1
}
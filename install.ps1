$ErrorActionPreference = "Stop"

$Repo = if ($env:GZY_REPO) { $env:GZY_REPO } else { "frimpsss/gzy" }
$Version = if ($env:GZY_VERSION) { $env:GZY_VERSION } else { "latest" }
$BinDir = if ($env:GZY_BIN_DIR) { $env:GZY_BIN_DIR } else { Join-Path $HOME "bin" }
$Arch = if ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture -eq "Arm64") { "arm64" } else { "x86_64" }
$Artifact = "gzy_Windows_$Arch.zip"

if ($Version -eq "latest") {
  $Url = "https://github.com/$Repo/releases/latest/download/$Artifact"
} else {
  $Url = "https://github.com/$Repo/releases/download/$Version/$Artifact"
}

$Temp = New-Item -ItemType Directory -Path ([System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), [System.Guid]::NewGuid().ToString()))
try {
  New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
  $ZipPath = Join-Path $Temp.FullName $Artifact
  Invoke-WebRequest -Uri $Url -OutFile $ZipPath
  Expand-Archive -Path $ZipPath -DestinationPath $Temp.FullName -Force
  Copy-Item (Join-Path $Temp.FullName "gzy.exe") (Join-Path $BinDir "gzy.exe") -Force
  Write-Host "gzy installed to $BinDir\gzy.exe"
  if (($env:PATH -split ";") -notcontains $BinDir) {
    Write-Host "Add this directory to PATH: $BinDir"
  }
} finally {
  Remove-Item -Recurse -Force $Temp.FullName
}

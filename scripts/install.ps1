$ErrorActionPreference = 'Stop'

$repo = 'rkirkendall/nano-agent'
$bin = 'nano-agent.exe'

$latest = (Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest").tag_name
$arch = if ($env:PROCESSOR_ARCHITECTURE -match 'ARM64') { 'arm64' } else { 'amd64' }
$url = "https://github.com/$repo/releases/download/$latest/nano-agent_windows_${arch}.zip"

$tmp = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath()) -Name (New-Guid)
Invoke-WebRequest -Uri $url -OutFile "$tmp\nano-agent.zip"
Expand-Archive -Path "$tmp\nano-agent.zip" -DestinationPath $tmp
$dest = "$env:ProgramFiles\nano-agent"
New-Item -ItemType Directory -Force -Path $dest | Out-Null
Copy-Item "$tmp\nano-agent\nano-agent.exe" "$dest\nano-agent.exe" -Force
[Environment]::SetEnvironmentVariable('Path', $env:Path + ";$dest", [EnvironmentVariableTarget]::Machine)
Write-Host "Installed nano-agent to $dest\nano-agent.exe"


# nano-agent
<img src="header.png" alt="nano-agent header" width="50%">


nano-agent is a CLI for generating, editing and compositing images using Google's `gemini-2.5-flash-image-preview` ("Nano Banana") model.

## Features
- Generate images from a prompt directly from the terminal
- Pass in images to edit or composite with a prompt
- Support for reusable prompt fragments via `-f/--fragment`
- Support for critique-improve feedback loops via `-cl/--critique-loops`

## Install

### macOS (Homebrew):
```
brew tap rkirkendall/tap
brew install rkirkendall/tap/nano-agent
```

### Windows (PowerShell, one‑liner; adds to PATH for current user):
(lol idk if this actually works but try it?)
```powershell
iwr https://raw.githubusercontent.com/rkirkendall/nano-agent/main/scripts/install.ps1 -UseB | iex; 
$dest = "$env:ProgramFiles\nano-agent"; 
if ($env:Path -notlike "*$dest*") { [Environment]::SetEnvironmentVariable('Path', $env:Path + ';' + $dest, 'User') }; 
$env:Path = [Environment]::GetEnvironmentVariable('Path','User') + ';' + [Environment]::GetEnvironmentVariable('Path','Machine')
```

Set `GEMINI_API_KEY` in your environment (e.g., in a local `.env` or your shell). If you don’t have one yet, get a key from [Google AI Studio](https://aistudio.google.com/apikey).

```
export GEMINI_API_KEY=your_key_here
```

## Usage

- Minimal text-to-image:
```
nano-agent -p "Create a picture of a nano banana dish in a fancy restaurant with a Gemini theme"
```

- Multi-image composition (image-in → image-out):
```
nano-agent -p "Blend these into a single collage in a sleek editorial style" img1.png img2.png img3.jpg
```

- Character + setting composite (scene directions in prompt):
```
nano-agent -p "Place the character on the left third, facing right; blend lighting to match the warm sunset; keep the character's outfit unchanged" character.png setting.png
```

- With reusable fragments:
```
nano-agent -p "Studio portrait" -f fragments/photorealism.txt fragments/portrait_lighting.txt
```

- Run critique-improve loops (Python parity; `-cl` is supported):
```
nano-agent -p "Portrait in an office" office_base.png -cl 3
# Iteration copies are saved as: outputs/office_improved_1.png, _2.png, _3.png
```

- Custom output paths (optional `-o`, defaults to output.png):
```
nano-agent -p "Refine this scene" base1.png base2.png -o runs/pass1.png -cl 2
# Writes to runs/pass1.png and copies to runs/outputs/pass1_improved_1.png, _2.png
```

## Version & updates
- Print version: `nano-agent -v` (or `--version`)
- macOS updates follow Homebrew: `brew update && brew upgrade rkirkendall/tap/nano-agent`

## Build from source (optional)
Prerequisites: Go 1.21+

```
git clone https://github.com/rkirkendall/nano-agent.git
cd nano-agent
go build ./cmd/nano-agent
./nano-agent --help
```

Auto-update: on startup, the CLI checks GitHub for a newer version and prints an upgrade hint if available.

## Configuration
- Environment variable: `GEMINI_API_KEY` (required)
- `.env` is auto-read if present; existing environment vars are not overridden.


## Uninstall

### macOS (Homebrew):
```
brew uninstall rkirkendall/tap/nano-agent
brew untap rkirkendall/tap
```

### Windows (PowerShell):
```powershell
$dest = "$env:ProgramFiles\nano-agent"
if (Test-Path "$dest\nano-agent.exe") { Remove-Item "$dest\nano-agent.exe" -Force }
# Remove user PATH entry if present (optional):
$userPath = [Environment]::GetEnvironmentVariable('Path','User').Split(';') | Where-Object { $_ -ne $dest }
[Environment]::SetEnvironmentVariable('Path', ($userPath -join ';'), 'User')
```

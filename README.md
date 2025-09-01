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

- Generate a comic-styled panel from a prompt (uses the comic style fragment):
```
nano-agent -p "Single-panel comic: a playful banana detective in an office" \
  -f examples/comic/fragments/comic-style.txt \
  -o examples/comic/panels/panel_prompt.png
```

- Compose a character into a place to form a panel:
```
nano-agent -p "Compose as a left-facing medium shot; keep character proportions" \
  examples/comic/characters/dan.png \
  examples/comic/place/office.png \
  -f examples/comic/fragments/comic-style.txt \
  -o examples/comic/panels/panel_dan_office.png
```

- Multi-image composition (two characters + a place):
```
nano-agent -p "Two-character panel; Dan on left, Barly on right; match lighting" \
  examples/comic/characters/dan.png \
  examples/comic/characters/barly.png \
  examples/comic/place/office.png \
  -f examples/comic/fragments/comic-style.txt \
  -o examples/comic/panels/panel_dan_barly_office.png
```

- With reusable fragments (style only):
```
nano-agent -p "Comic panel: Dan delivering a punchline" \
  -f examples/comic/fragments/comic-style.txt \
  -o examples/comic/panels/panel_gag.png
```

- Run critique-improve loops on a produced panel (`-cl` is supported):
```
nano-agent -p "Tighten line work and add stronger rim light" \
  examples/comic/panels/panel_dan_office.png \
  -cl 3 \
  -o examples/comic/panels/panel_dan_office_v2.png
# Iterations are saved to: examples/comic/panels/outputs/panel_dan_office_v2_improved_1.png, _2.png, _3.png
```

- Custom output path for multi-panel inputs (defaults to output.png if omitted):
```
nano-agent -p "Three-panel page, consistent style and palette" \
  examples/comic/panels/one.png \
  examples/comic/panels/two.png \
  examples/comic/panels/three.png \
  -f examples/comic/fragments/comic-style.txt \
  -o examples/comic/panels/page_comp.png
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

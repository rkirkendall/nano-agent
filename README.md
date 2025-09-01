## nano-agent

Nano Agent is a cross-platform CLI for generating and iteratively improving images with Google's Gemini models using critique-improve loops and emphasis on actionable follow-up guidance.

### Features
- Generate from zero or more input images and a prompt
- Reusable prompt fragments via `-f/--fragment`
- Iterative critique-improve loops with escalation of unresolved items
- Clear severity tags: `[CRITICAL — persisted]`, `[MAJOR]`, `[MINOR]`
- Imperative targeted actions list to drive the next iteration

### Quick start
Prerequisites: Go 1.21+

```
git clone https://github.com/rickycodes/nano-agent.git
cd nano-agent
go build ./cmd/nano-agent
./nano-agent --help
```

Set `GEMINI_API_KEY` in your environment.

### Install (early phase options)
- Homebrew (planned via GoReleaser tap)
- Scoop (planned)
- Direct download from GitHub Releases (planned)

For now:
```
go install github.com/rickycodes/nano-agent/cmd/nano-agent@latest
```

### Usage
```
nano-agent generate --prompt "Ultra-realistic product photo of a ceramic mug" -o output.png
nano-agent critique --image output.png --prompt "Ultra-realistic product photo of a ceramic mug"
nano-agent loop --prompt "Portrait of a person in an office" -o output.png -c 3
```

### Configuration
Environment variable: `GEMINI_API_KEY` for the Gemini API key.

### Status
This is an early Go port of the Python prototype. Roadmap includes complete Gemini integration, GoReleaser packaging, and broader test coverage.



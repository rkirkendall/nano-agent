![nano-agent header](header.png)

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

Set `GEMINI_API_KEY` in your environment (e.g., in a local `.env` or your shell).

```
export GEMINI_API_KEY=your_key_here
```

### Usage

- Minimal text-to-image:
```
./nano-agent --prompt "Create a picture of a nano banana dish in a fancy restaurant with a Gemini theme"
```

- Multi-image composition (image-in → image-out):
```
./nano-agent --prompt "Blend these into a single collage in a sleek editorial style" img1.png img2.png img3.jpg
```

- Character + setting composite (scene directions in prompt):
```
./nano-agent --prompt "Place the character on the left third, facing right; blend lighting to match the warm sunset; keep the character's outfit unchanged" character.png setting.png
```

- With reusable fragments:
```
./nano-agent --prompt "Studio portrait" -f fragments/photorealism.txt fragments/portrait_lighting.txt
```

- Run critique-improve loops (Python parity; `-cl` is supported):
```
./nano-agent --prompt "Portrait in an office" office_base.png -cl 3
# Iteration copies are saved as: outputs/office_improved_1.png, _2.png, _3.png
```

- Custom output paths (optional `-o`, defaults to output.png):
```
./nano-agent --prompt "Refine this scene" base1.png base2.png -o runs/pass1.png -cl 2
# Writes to runs/pass1.png and copies to runs/outputs/pass1_improved_1.png, _2.png
```

Notes:
- You can also provide images via `--images`, but positional args are preferred.
- The tool overwrites the specified `-o` file each loop and writes iteration copies under an adjacent `outputs/` directory.

### Configuration
- Environment variable: `GEMINI_API_KEY` (required)
- `.env` is auto-read if present; existing environment vars are not overridden.

### Status
This is an early Go port of the Python prototype. Roadmap includes GoReleaser packaging and expanded tests.



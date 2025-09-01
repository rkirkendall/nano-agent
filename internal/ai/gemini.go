package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
)

func loadEnvIfMissing() {
	b, err := os.ReadFile(".env")
	if err == nil {
		lines := strings.Split(string(b), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if i := strings.IndexByte(line, '='); i > 0 {
				k := strings.TrimSpace(line[:i])
				v := strings.TrimSpace(line[i+1:])
				if _, exists := os.LookupEnv(k); !exists && k != "" {
					_ = os.Setenv(k, v)
				}
			}
		}
	}
}

// ensureAPIKey enforces GEMINI_API_KEY usage and configures the client env.
func ensureAPIKey() error {
	loadEnvIfMissing()
	k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if k == "" {
		return errors.New("GEMINI_API_KEY is not set")
	}
	// Force the genai client to use GEMINI_API_KEY by mapping it and removing GOOGLE_API_KEY.
	_ = os.Unsetenv("GOOGLE_API_KEY")
	_ = os.Setenv("GOOGLE_API_KEY", k)
	return nil
}

// GenerateImage using google.golang.org/genai for text-to-image and image+text-to-image
func GenerateImage(ctx context.Context, model string, imagePaths []string, prompt string, fragments []string) ([]byte, error) {
	if err := ensureAPIKey(); err != nil {
		return nil, err
	}
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, err
	}

	// If no input images, use simple Text contents flow
	if len(imagePaths) == 0 {
		var contents []*genai.Content
		if s := strings.TrimSpace(prompt); s != "" {
			contents = append(contents, genai.Text(s)...)
		}
		for _, f := range fragments {
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			if s := strings.TrimSpace(string(b)); s != "" {
				contents = append(contents, genai.Text(s)...)
			}
		}
		res, err := client.Models.GenerateContent(ctx, model, contents, nil)
		if err != nil {
			return nil, err
		}
		if len(res.Candidates) > 0 && res.Candidates[0].Content != nil {
			for _, part := range res.Candidates[0].Content.Parts {
				if part.InlineData != nil && len(part.InlineData.Data) > 0 {
					return part.InlineData.Data, nil
				}
			}
		}
		return nil, errors.New("no image returned by model")
	}

	// With input images: build Parts then wrap into a single Content
	var parts []*genai.Part
	if s := strings.TrimSpace(prompt); s != "" {
		parts = append(parts, genai.NewPartFromText(s))
	}
	for _, f := range fragments {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			parts = append(parts, genai.NewPartFromText(s))
		}
	}
	for _, p := range imagePaths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		mime := "image/png"
		switch strings.ToLower(filepath.Ext(p)) {
		case ".jpg", ".jpeg":
			mime = "image/jpeg"
		case ".webp":
			mime = "image/webp"
		case ".gif":
			mime = "image/gif"
		}
		parts = append(parts, &genai.Part{InlineData: &genai.Blob{MIMEType: mime, Data: b}})
	}
	contents := []*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}
	res, err := client.Models.GenerateContent(ctx, model, contents, nil)
	if err != nil {
		return nil, err
	}
	if len(res.Candidates) > 0 && res.Candidates[0].Content != nil {
		for _, part := range res.Candidates[0].Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				return part.InlineData.Data, nil
			}
		}
	}
	return nil, errors.New("no image returned by model")
}

// GenerateCritique using google.golang.org/genai: text out with image input
func GenerateCritique(ctx context.Context, model string, imagePath string, originalPrompt string, fragments []string, previousCritique string) (string, error) {
	if err := ensureAPIKey(); err != nil {
		return "", err
	}
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return "", err
	}

	instruction := "You are an expert image forensics and quality reviewer. Given the provided image and the original text prompt, write a concise critique.\n\n- Identify visual artifacts, anatomical errors, texture inconsistencies, lighting/shadow mismatches, reflections, perspective errors, or other signs of AI generation or poor editing.\n- Point out mismatches between the image and the prompt.\n- Provide actionable suggestions for improving realism and prompt phrasing.\n\nReturn plain text."
	if strings.TrimSpace(previousCritique) != "" {
		instruction += "\n\nFollow-on critique guidance:\n- Be especially mindful of 'fuzzy' generation artifacts...\n- Compare the current image against the previous critique and indicate what improved versus what remains unresolved.\n- If an issue noted previously remains visible, escalate it: rewrite the item with imperative wording and tag [CRITICAL — persisted]; include locations.\n- Order unresolved items by severity: [CRITICAL — persisted], [MAJOR], [MINOR].\n- Output two checklists: improvements and remaining issues, then 2-4 sentences of targeted guidance, and a short 'Targeted actions to apply now' list."
	}

	imgBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(imagePath)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}
	parts := []*genai.Part{
		genai.NewPartFromText(instruction),
		genai.NewPartFromText(fmt.Sprintf("Original prompt:\n%s", originalPrompt)),
		{InlineData: &genai.Blob{MIMEType: mime, Data: imgBytes}},
	}
	if strings.TrimSpace(previousCritique) != "" {
		parts = append(parts, genai.NewPartFromText("Previous critique (for follow-on precision):\n"+previousCritique))
	}
	for _, f := range fragments {
		b, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			parts = append(parts, genai.NewPartFromText(s))
		}
	}
	contents := []*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}
	resp, err := client.Models.GenerateContent(ctx, model, contents, nil)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, p := range resp.Candidates[0].Content.Parts {
			if p.Text != "" {
				out.WriteString(p.Text)
			}
		}
	}
	if s := strings.TrimSpace(out.String()); s != "" {
		return s, nil
	}
	return "", errors.New("no text returned by model")
}

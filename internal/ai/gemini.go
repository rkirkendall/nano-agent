package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/rkirkendall/nano-agent/internal/critique"
	"github.com/rkirkendall/nano-agent/internal/generate"
	"google.golang.org/genai"
)

// ============================
// Constants & configuration
// ============================

const (
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	defaultGeminiImageModel  = "models/gemini-2.5-flash-image-preview"
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

func isOpenRouterEnabled() bool {
	v := strings.TrimSpace(os.Getenv("USE_OPENROUTER"))
	return v == "1" || strings.EqualFold(v, "true")
}

func ensureOpenRouterKey() error {
	loadEnvIfMissing()
	if !isOpenRouterEnabled() {
		return nil
	}
	k := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if k == "" {
		return errors.New("OPENROUTER_API_KEY is required when USE_OPENROUTER=1")
	}
	return nil
}

func mapModelForOpenRouter(model string) string {
	// Environment override takes precedence to allow ops-configured models
	if env := strings.TrimSpace(os.Getenv("OPENROUTER_MODEL")); env != "" {
		return env
	}
	m := strings.TrimSpace(model)
	if m == "" {
		return "google/gemini-2.5-flash-image-preview:free"
	}
	if strings.Contains(m, "/") {
		return m
	}
	if strings.Contains(m, ":") {
		return "google/" + m
	}
	return "google/" + m
}

// mapModelForGemini normalizes model names for the native Gemini SDK.
// Accepts inputs like:
//   - "gemini-2.5-flash-image-preview:free"
//   - "gemini-2.5-flash-image-preview"
//   - "google/gemini-2.5-flash-image-preview:free"
//   - "models/gemini-2.5-flash-image-preview"
//
// and returns a resource name like:
//   - "models/gemini-2.5-flash-image-preview"
func mapModelForGemini(model string) string {
	m := strings.TrimSpace(model)
	if m == "" {
		return defaultGeminiImageModel
	}
	// already a full resource name
	if strings.HasPrefix(m, "models/") {
		return m
	}
	// strip provider namespace if present
	m = strings.TrimPrefix(m, "google/")
	// drop any trailing ":..." variant suffix
	if i := strings.IndexByte(m, ':'); i >= 0 {
		m = m[:i]
	}
	return "models/" + m
}

func newOpenRouterClient() openai.Client {
	return openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		option.WithBaseURL(defaultOpenRouterBaseURL),
	)
}

func getOpenRouterBaseURL() string {
	b := strings.TrimSpace(os.Getenv("OPENROUTER_BASE_URL"))
	if b != "" {
		return b
	}
	return defaultOpenRouterBaseURL
}

func httpJSON(client openai.Client, ctx context.Context, path string, body any) (map[string]any, error) {
	// Use direct HTTP request to avoid SDK path quirks
	path = strings.TrimLeft(path, "/")
	base := getOpenRouterBaseURL()
	url := strings.TrimRight(base, "/") + "/" + path

	breq, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(breq)))
	if err != nil {
		return nil, err
	}
	k := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	req.Header.Set("Authorization", "Bearer "+k)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	referer := strings.TrimSpace(os.Getenv("OPENROUTER_SITE"))
	if referer == "" {
		referer = "http://localhost"
	}
	req.Header.Set("HTTP-Referer", referer)
	title := strings.TrimSpace(os.Getenv("OPENROUTER_TITLE"))
	if title == "" {
		title = "nano-agent"
	}
	req.Header.Set("X-Title", title)
	req.Header.Set("User-Agent", "nano-agent/1.0 (+github.com/rkirkendall/nano-agent)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if os.Getenv("OPENROUTER_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "DEBUG openrouter POST %s status=%v auth=%t\n", url, resp.Status, strings.TrimSpace(k) != "")
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if os.Getenv("OPENROUTER_DEBUG") == "1" {
		preview := string(b)
		if len(preview) > 4096 {
			preview = preview[:4096] + "... [truncated]"
		}
		fmt.Fprintf(os.Stderr, "DEBUG openrouter BODY %s\n", preview)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		preview := string(b)
		if len(preview) > 2048 {
			preview = preview[:2048] + "... [truncated]"
		}
		return nil, fmt.Errorf("openrouter decode failed: %w; status=%d; body=%s", err, resp.StatusCode, preview)
	}
	return out, nil
}

func toDataURL(mime string, b []byte) string {
	return fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(b))
}

func guessMIME(p string) string {
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(p)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}
	return mime
}

func parseImageFromResponsesJSON(m map[string]any) ([]byte, error) {
	// Responses API shape
	if outArr, ok := m["output"].([]any); ok {
		for _, item := range outArr {
			obj, _ := item.(map[string]any)
			if obj == nil {
				continue
			}
			contentArr, _ := obj["content"].([]any)
			for _, c := range contentArr {
				cobj, _ := c.(map[string]any)
				if cobj == nil {
					continue
				}
				t, _ := cobj["type"].(string)
				if t == "output_image" || t == "image" || t == "image_url" {
					if img, ok := cobj["image"].(map[string]any); ok {
						if s, ok := img["b64_json"].(string); ok && s != "" {
							return base64.StdEncoding.DecodeString(s)
						}
						if s, ok := img["b64"].(string); ok && s != "" {
							return base64.StdEncoding.DecodeString(s)
						}
						if u, ok := img["url"].(string); ok && strings.HasPrefix(u, "data:") {
							if i := strings.IndexByte(u, ','); i > 0 {
								return base64.StdEncoding.DecodeString(u[i+1:])
							}
						}
					}
					if s, ok := cobj["b64_json"].(string); ok && s != "" {
						return base64.StdEncoding.DecodeString(s)
					}
				}
			}
		}
	}
	// Images API style
	if dataArr, ok := m["data"].([]any); ok {
		for _, d := range dataArr {
			dobj, _ := d.(map[string]any)
			if dobj == nil {
				continue
			}
			if s, ok := dobj["b64_json"].(string); ok && s != "" {
				return base64.StdEncoding.DecodeString(s)
			}
			if u, ok := dobj["url"].(string); ok && strings.HasPrefix(u, "data:") {
				if i := strings.IndexByte(u, ','); i > 0 {
					return base64.StdEncoding.DecodeString(u[i+1:])
				}
			}
		}
	}
	return nil, errors.New("no image returned by model")
}

func parseImageFromChatJSON(m map[string]any) ([]byte, error) {
	if choices, ok := m["choices"].([]any); ok && len(choices) > 0 {
		ch, _ := choices[0].(map[string]any)
		if ch != nil {
			if msg, ok := ch["message"].(map[string]any); ok {
				// Some providers return images in message.images (not within content parts)
				if imgs, _ := msg["images"].([]any); len(imgs) > 0 {
					if im0, _ := imgs[0].(map[string]any); im0 != nil {
						// image_url wrapper
						if iu, ok := im0["image_url"].(map[string]any); ok {
							if u, _ := iu["url"].(string); strings.HasPrefix(u, "data:") {
								if i := strings.IndexByte(u, ','); i > 0 {
									return base64.StdEncoding.DecodeString(u[i+1:])
								}
							}
						}
						// nested image map
						if imgMap, ok := im0["image"].(map[string]any); ok {
							if s, ok := imgMap["b64_json"].(string); ok && s != "" {
								return base64.StdEncoding.DecodeString(s)
							}
							if s, ok := imgMap["b64"].(string); ok && s != "" {
								return base64.StdEncoding.DecodeString(s)
							}
							if u, ok := imgMap["url"].(string); ok && strings.HasPrefix(u, "data:") {
								if i := strings.IndexByte(u, ','); i > 0 {
									return base64.StdEncoding.DecodeString(u[i+1:])
								}
							}
						}
					}
				}
				// content can be string or array of parts
				if parts, ok := msg["content"].([]any); ok {
					for _, p := range parts {
						pobj, _ := p.(map[string]any)
						if pobj == nil {
							continue
						}
						// Common shapes: image_url { url: data:... }, b64_json string, or output_image { image: { b64_json|b64|url } }
						if t, _ := pobj["type"].(string); t == "image_url" || t == "image" || t == "output_image" {
							// Nested image map
							if img, ok := pobj["image"].(map[string]any); ok {
								if s, ok := img["b64_json"].(string); ok && s != "" {
									return base64.StdEncoding.DecodeString(s)
								}
								if s, ok := img["b64"].(string); ok && s != "" {
									return base64.StdEncoding.DecodeString(s)
								}
								if u, ok := img["url"].(string); ok && strings.HasPrefix(u, "data:") {
									if i := strings.IndexByte(u, ','); i > 0 {
										return base64.StdEncoding.DecodeString(u[i+1:])
									}
								}
							}
							// image_url wrapper
							if iu, ok := pobj["image_url"].(map[string]any); ok {
								if u, _ := iu["url"].(string); strings.HasPrefix(u, "data:") {
									if i := strings.IndexByte(u, ','); i > 0 {
										return base64.StdEncoding.DecodeString(u[i+1:])
									}
								}
							}
							// direct b64 on the part
							if s, ok := pobj["b64_json"].(string); ok && s != "" {
								return base64.StdEncoding.DecodeString(s)
							}
						}
					}
				}
				// some providers embed a single data URL string
				if s, ok := msg["content"].(string); ok && strings.HasPrefix(s, "data:") {
					if i := strings.IndexByte(s, ','); i > 0 {
						return base64.StdEncoding.DecodeString(s[i+1:])
					}
				}
			}
		}
	}
	return nil, errors.New("no image returned by model")
}

func parseTextFromChatJSON(m map[string]any) (string, error) {
	// OpenAI chat completions
	if choices, ok := m["choices"].([]any); ok && len(choices) > 0 {
		ch, _ := choices[0].(map[string]any)
		if ch != nil {
			if msg, ok := ch["message"].(map[string]any); ok {
				if s, ok := msg["content"].(string); ok && strings.TrimSpace(s) != "" {
					return s, nil
				}
				if parts, ok := msg["content"].([]any); ok {
					var sb strings.Builder
					for _, p := range parts {
						pobj, _ := p.(map[string]any)
						if pobj == nil {
							continue
						}
						if t, _ := pobj["type"].(string); t == "text" || t == "output_text" {
							if s, _ := pobj["text"].(string); s != "" {
								sb.WriteString(s)
							}
						}
					}
					if strings.TrimSpace(sb.String()) != "" {
						return sb.String(), nil
					}
				}
			}
		}
	}
	// Responses API
	if outArr, ok := m["output"].([]any); ok {
		var sb strings.Builder
		for _, item := range outArr {
			obj, _ := item.(map[string]any)
			if obj == nil {
				continue
			}
			contentArr, _ := obj["content"].([]any)
			for _, c := range contentArr {
				cobj, _ := c.(map[string]any)
				if cobj == nil {
					continue
				}
				if t, _ := cobj["type"].(string); t == "output_text" || t == "text" {
					if s, _ := cobj["text"].(string); s != "" {
						sb.WriteString(s)
					}
				}
			}
		}
		if strings.TrimSpace(sb.String()) != "" {
			return sb.String(), nil
		}
	}
	return "", errors.New("no text returned by model")
}

// ensureAPIKey enforces GEMINI_API_KEY usage and configures the client env.
func ensureAPIKey() error {
	loadEnvIfMissing()
	if isOpenRouterEnabled() {
		return ensureOpenRouterKey()
	}
	k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if k == "" {
		return errors.New("GEMINI_API_KEY is not set; get one at https://aistudio.google.com/apikey and export GEMINI_API_KEY before running")
	}
	_ = os.Unsetenv("GOOGLE_API_KEY")
	_ = os.Setenv("GOOGLE_API_KEY", k)
	return nil
}

// GenerateImage routes to OpenRouter or the Gemini SDK to produce an image from an optional
// set of input images plus a text prompt and fragments. Returns PNG bytes on success.
func GenerateImage(ctx context.Context, model string, imagePaths []string, prompt string, fragments []string) ([]byte, error) {
	if err := ensureAPIKey(); err != nil {
		return nil, err
	}
	if isOpenRouterEnabled() {
		client := newOpenRouterClient()
		effPrompt := generate.BuildEffectivePrompt(prompt, fragments)
		baseContent := make([]any, 0, 1+len(imagePaths))
		if s := strings.TrimSpace(effPrompt); s != "" {
			baseContent = append(baseContent, map[string]any{"type": "text", "text": s})
		}
		withImages := make([]any, 0, len(baseContent)+len(imagePaths))
		withImages = append(withImages, baseContent...)
		for _, p := range imagePaths {
			bimg, rerr := os.ReadFile(p)
			if rerr == nil && len(bimg) > 0 {
				withImages = append(withImages, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": toDataURL(guessMIME(p), bimg)},
				})
			}
		}

		// Helper to call chat/completions and parse an image
		doChat := func(content []any) ([]byte, error) {
			req := map[string]any{
				"model":    mapModelForOpenRouter(model),
				"messages": []any{map[string]any{"role": "user", "content": content}},
			}
			m, err := httpJSON(client, ctx, "chat/completions", req)
			if err != nil {
				return nil, err
			}
			if errObj, ok := m["error"].(map[string]any); ok {
				if msg, _ := errObj["message"].(string); strings.TrimSpace(msg) != "" {
					return nil, errors.New(msg)
				}
				return nil, errors.New("OpenRouter returned an error during image generation")
			}
			if choices, ok := m["choices"].([]any); ok && len(choices) > 0 {
				if ch, _ := choices[0].(map[string]any); ch != nil {
					if msg, _ := ch["message"].(map[string]any); msg != nil {
						if imgs, _ := msg["images"].([]any); len(imgs) > 0 {
							if im0, _ := imgs[0].(map[string]any); im0 != nil {
								if iu, _ := im0["image_url"].(map[string]any); iu != nil {
									if u, _ := iu["url"].(string); strings.HasPrefix(u, "data:") {
										if i := strings.IndexByte(u, ','); i > 0 {
											return base64.StdEncoding.DecodeString(u[i+1:])
										}
									}
								}
							}
						}
					}
				}
			}
			if img, err2 := parseImageFromChatJSON(m); err2 == nil && len(img) > 0 {
				return img, nil
			}
			// Surface assistant text when no image is returned to expose provider errors
			if txt, _ := parseTextFromChatJSON(m); strings.TrimSpace(txt) != "" {
				trim := txt
				if len(trim) > 512 {
					trim = trim[:512] + "..."
				}
				return nil, fmt.Errorf("no image returned by model: %s", strings.TrimSpace(trim))
			}
			return nil, errors.New("no image returned by model")
		}

		// Try with images (up to 2 attempts), then as last resort text-only
		var lastErr error
		for attempt := 1; attempt <= 2; attempt++ {
			if os.Getenv("OPENROUTER_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "DEBUG openrouter attempt %d with images=%t\n", attempt, len(imagePaths) > 0)
			}
			if len(imagePaths) > 0 {
				if img, err := doChat(withImages); err == nil && len(img) > 0 {
					return img, nil
				} else if err != nil {
					lastErr = err
				}
			}
		}
		// Last attempt: text-only
		if img, err := doChat(baseContent); err == nil && len(img) > 0 {
			return img, nil
		} else if err != nil {
			lastErr = err
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, errors.New("no image returned by model")
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
		res, err := client.Models.GenerateContent(ctx, mapModelForGemini(model), contents, nil)
		if err != nil {
			return nil, err
		}
		if len(res.Candidates) > 0 && res.Candidates[0].Content != nil {
			var textOut strings.Builder
			for _, part := range res.Candidates[0].Content.Parts {
				if part.InlineData != nil && len(part.InlineData.Data) > 0 {
					return part.InlineData.Data, nil
				}
				if part.Text != "" {
					textOut.WriteString(part.Text)
				}
			}
			if s := strings.TrimSpace(textOut.String()); s != "" {
				if len(s) > 512 {
					s = s[:512] + "..."
				}
				return nil, fmt.Errorf("no image returned by model: %s", s)
			}
		}
		return nil, errors.New("no image returned by model")
	}

	// With input images: build Parts then wrap into a single Content
	var partsGen []*genai.Part
	if s := strings.TrimSpace(prompt); s != "" {
		partsGen = append(partsGen, genai.NewPartFromText(s))
	}
	for _, f := range fragments {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			partsGen = append(partsGen, genai.NewPartFromText(s))
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
		partsGen = append(partsGen, &genai.Part{InlineData: &genai.Blob{MIMEType: mime, Data: b}})
	}
	contents := []*genai.Content{genai.NewContentFromParts(partsGen, genai.RoleUser)}
	res, err := client.Models.GenerateContent(ctx, mapModelForGemini(model), contents, nil)
	if err != nil {
		return nil, err
	}
	if len(res.Candidates) > 0 && res.Candidates[0].Content != nil {
		var textOut strings.Builder
		for _, part := range res.Candidates[0].Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				return part.InlineData.Data, nil
			}
			if part.Text != "" {
				textOut.WriteString(part.Text)
			}
		}
		if s := strings.TrimSpace(textOut.String()); s != "" {
			if len(s) > 512 {
				s = s[:512] + "..."
			}
			return nil, fmt.Errorf("no image returned by model: %s", s)
		}
	}
	return nil, errors.New("no image returned by model")
}

// GenerateCritique produces actionable critique text for a given image using OpenRouter or
// the Gemini SDK. It includes the original prompt and optional input reference images.
func GenerateCritique(ctx context.Context, model string, imagePath string, originalPrompt string, fragments []string, inputImagePaths []string) (string, error) {
	if err := ensureAPIKey(); err != nil {
		return "", err
	}
	if isOpenRouterEnabled() {
		client := newOpenRouterClient()
		instruction := critique.BuildCritiqueInstruction()
		// Build chat/completions style message with text + image_url parts
		parts := make([]any, 0, 4)
		parts = append(parts, map[string]any{"type": "text", "text": instruction})
		if s := strings.TrimSpace(originalPrompt); s != "" {
			parts = append(parts, map[string]any{"type": "text", "text": fmt.Sprintf("Original prompt:\n%s", s)})
		}
		imgBytes, err := os.ReadFile(imagePath)
		if err != nil {
			return "", err
		}
		parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": toDataURL(guessMIME(imagePath), imgBytes)}})
		if len(inputImagePaths) > 0 {
			parts = append(parts, map[string]any{"type": "text", "text": "Original input images for reference:"})
			for _, pth := range inputImagePaths {
				b, err := os.ReadFile(pth)
				if err != nil {
					return "", err
				}
				parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": toDataURL(guessMIME(pth), b)}})
			}
		}
		for _, f := range fragments {
			b, err := os.ReadFile(f)
			if err != nil {
				return "", err
			}
			if s := strings.TrimSpace(string(b)); s != "" {
				parts = append(parts, map[string]any{"type": "text", "text": s})
			}
		}
		req := map[string]any{
			"model": mapModelForOpenRouter(model),
			"messages": []any{
				map[string]any{
					"role":    "user",
					"content": parts,
				},
			},
		}
		m, err := httpJSON(client, ctx, "chat/completions", req)
		if err != nil {
			return "", err
		}
		// Propagate OpenRouter error body (no fallbacks)
		if errObj, ok := m["error"].(map[string]any); ok {
			if msg, _ := errObj["message"].(string); strings.TrimSpace(msg) != "" {
				return "", errors.New(msg)
			}
			return "", errors.New("OpenRouter returned an error during critique")
		}
		return parseTextFromChatJSON(m)
	}
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return "", err
	}

	instruction := critique.BuildCritiqueInstruction()

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
	// Attach original input images for context, if provided
	if len(inputImagePaths) > 0 {
		parts = append(parts, genai.NewPartFromText("Original input images for reference:"))
		for _, pth := range inputImagePaths {
			b, err := os.ReadFile(pth)
			if err != nil {
				return "", err
			}
			im := "image/png"
			switch strings.ToLower(filepath.Ext(pth)) {
			case ".jpg", ".jpeg":
				im = "image/jpeg"
			case ".webp":
				im = "image/webp"
			case ".gif":
				im = "image/gif"
			}
			parts = append(parts, &genai.Part{InlineData: &genai.Blob{MIMEType: im, Data: b}})
		}
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
	resp, err := client.Models.GenerateContent(ctx, mapModelForGemini(model), contents, nil)
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

// ============================
// Threaded image generation
// ============================

// ImageThread maintains a conversation history for iterative image generation so
// that critique prompts can be appended to the original generation thread.
type ImageThread struct {
	model         string
	useOpenRouter bool
	orClient      openai.Client
	orMessages    []any
	geminiClient  *genai.Client
	geminiHistory []*genai.Content
}

// StartImageThreadAndGenerate creates a new image generation thread with the initial
// prompt, fragments, and optional input images, generates an image, and returns the
// thread along with the generated PNG bytes.
func StartImageThreadAndGenerate(ctx context.Context, model string, imagePaths []string, prompt string, fragments []string) (*ImageThread, []byte, error) {
	if err := ensureAPIKey(); err != nil {
		return nil, nil, err
	}
	thread := &ImageThread{model: model, useOpenRouter: isOpenRouterEnabled()}
	if thread.useOpenRouter {
		thread.orClient = newOpenRouterClient()
		// Build initial user message
		effPrompt := generate.BuildEffectivePrompt(prompt, fragments)
		parts := make([]any, 0, 1+len(imagePaths))
		if s := strings.TrimSpace(effPrompt); s != "" {
			parts = append(parts, map[string]any{"type": "text", "text": s})
		}
		for _, p := range imagePaths {
			bimg, rerr := os.ReadFile(p)
			if rerr == nil && len(bimg) > 0 {
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": toDataURL(guessMIME(p), bimg)},
				})
			}
		}
		thread.orMessages = []any{map[string]any{"role": "user", "content": parts}}
		img, text, err := thread.openRouterGenerate(ctx)
		if err != nil {
			return nil, nil, err
		}
		thread.appendOpenRouterAssistantMessage(img, text)
		return thread, img, nil
	}
	// Gemini path: create client and seed history with initial user content
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	thread.geminiClient = client
	var partsGen []*genai.Part
	if s := strings.TrimSpace(prompt); s != "" {
		partsGen = append(partsGen, genai.NewPartFromText(s))
	}
	for _, f := range fragments {
		b, rerr := os.ReadFile(f)
		if rerr != nil {
			return nil, nil, rerr
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			partsGen = append(partsGen, genai.NewPartFromText(s))
		}
	}
	for _, p := range imagePaths {
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil, nil, rerr
		}
		mime := guessMIME(p)
		partsGen = append(partsGen, &genai.Part{InlineData: &genai.Blob{MIMEType: mime, Data: b}})
	}
	thread.geminiHistory = []*genai.Content{genai.NewContentFromParts(partsGen, genai.RoleUser)}
	img, text, err := thread.geminiGenerate(ctx)
	if err != nil {
		return nil, nil, err
	}
	thread.appendGeminiAssistantMessage(img, text)
	return thread, img, nil
}

// AddUserMessageAndGenerate appends a new user message to the existing thread using the
// provided text (e.g., improvement prompt). If imagePath is non-empty, the image is
// also attached for model reference. Returns the newly generated PNG bytes.
func (t *ImageThread) AddUserMessageAndGenerate(ctx context.Context, text string, imagePath string) ([]byte, error) {
	if t.useOpenRouter {
		parts := make([]any, 0, 2)
		if s := strings.TrimSpace(text); s != "" {
			parts = append(parts, map[string]any{"type": "text", "text": s})
		}
		if strings.TrimSpace(imagePath) != "" {
			bimg, err := os.ReadFile(imagePath)
			if err == nil && len(bimg) > 0 {
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": toDataURL(guessMIME(imagePath), bimg)},
				})
			}
		}
		t.orMessages = append(t.orMessages, map[string]any{"role": "user", "content": parts})
		img, assistantText, err := t.openRouterGenerate(ctx)
		if err != nil {
			return nil, err
		}
		t.appendOpenRouterAssistantMessage(img, assistantText)
		return img, nil
	}
	// Gemini path
	var partsGen []*genai.Part
	if s := strings.TrimSpace(text); s != "" {
		partsGen = append(partsGen, genai.NewPartFromText(s))
	}
	if strings.TrimSpace(imagePath) != "" {
		b, err := os.ReadFile(imagePath)
		if err == nil && len(b) > 0 {
			mime := guessMIME(imagePath)
			partsGen = append(partsGen, &genai.Part{InlineData: &genai.Blob{MIMEType: mime, Data: b}})
		}
	}
	t.geminiHistory = append(t.geminiHistory, genai.NewContentFromParts(partsGen, genai.RoleUser))
	img, assistantText, err := t.geminiGenerate(ctx)
	if err != nil {
		return nil, err
	}
	t.appendGeminiAssistantMessage(img, assistantText)
	return img, nil
}

// openRouterGenerate performs a chat/completions call with accumulated messages and
// returns the generated image bytes and any assistant text.
func (t *ImageThread) openRouterGenerate(ctx context.Context) ([]byte, string, error) {
	req := map[string]any{
		"model":    mapModelForOpenRouter(t.model),
		"messages": t.orMessages,
	}
	m, err := httpJSON(t.orClient, ctx, "chat/completions", req)
	if err != nil {
		return nil, "", err
	}
	if errObj, ok := m["error"].(map[string]any); ok {
		if msg, _ := errObj["message"].(string); strings.TrimSpace(msg) != "" {
			return nil, "", errors.New(msg)
		}
		return nil, "", errors.New("OpenRouter returned an error during image generation")
	}
	img, imgErr := parseImageFromChatJSON(m)
	assistantText, _ := parseTextFromChatJSON(m)
	if imgErr != nil {
		return nil, assistantText, imgErr
	}
	return img, assistantText, nil
}

func (t *ImageThread) appendOpenRouterAssistantMessage(img []byte, assistantText string) {
	parts := make([]any, 0, 2)
	if strings.TrimSpace(assistantText) != "" {
		parts = append(parts, map[string]any{"type": "text", "text": assistantText})
	}
	if len(img) > 0 {
		parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": toDataURL("image/png", img)}})
	}
	if len(parts) > 0 {
		t.orMessages = append(t.orMessages, map[string]any{"role": "assistant", "content": parts})
	}
}

// geminiGenerate performs a multi-turn generation using the accumulated history and
// returns the generated image bytes and any assistant text.
func (t *ImageThread) geminiGenerate(ctx context.Context) ([]byte, string, error) {
	res, err := t.geminiClient.Models.GenerateContent(ctx, mapModelForGemini(t.model), t.geminiHistory, nil)
	if err != nil {
		return nil, "", err
	}
	var outText strings.Builder
	var outImg []byte
	if len(res.Candidates) > 0 && res.Candidates[0].Content != nil {
		for _, part := range res.Candidates[0].Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 && outImg == nil {
				outImg = part.InlineData.Data
			}
			if part.Text != "" {
				outText.WriteString(part.Text)
			}
		}
	}
	if len(outImg) == 0 {
		return nil, strings.TrimSpace(outText.String()), errors.New("no image returned by model")
	}
	return outImg, strings.TrimSpace(outText.String()), nil
}

func (t *ImageThread) appendGeminiAssistantMessage(img []byte, assistantText string) {
	var parts []*genai.Part
	if strings.TrimSpace(assistantText) != "" {
		parts = append(parts, genai.NewPartFromText(assistantText))
	}
	if len(img) > 0 {
		parts = append(parts, &genai.Part{InlineData: &genai.Blob{MIMEType: "image/png", Data: img}})
	}
	if len(parts) > 0 {
		t.geminiHistory = append(t.geminiHistory, genai.NewContentFromParts(parts, genai.RoleModel))
	}
}

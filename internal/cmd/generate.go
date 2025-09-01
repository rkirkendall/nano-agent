package cmd

import (
    "context"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    images []string
    prompt string
    fragments []string
    output string
)

func newGenerateCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "generate",
        Short: "Generate an image from zero or more input images and a prompt",
        Long:  "Generate an image guided by a text prompt and optional input images and reusable fragments.",
        RunE: func(cmd *cobra.Command, args []string) error {
            if strings.TrimSpace(prompt) == "" {
                return errors.New("--prompt is required")
            }
            if output == "" {
                output = "output.png"
            }
            // Placeholder: wire to internal/generate package
            fmt.Fprintf(cmd.OutOrStdout(), "Generating to %s using model %s...\n", output, viper.GetString("model"))
            _ = context.Background()
            _ = images
            _ = fragments
            // TODO: call generate pipeline and save file
            return nil
        },
        Example: `nano-agent generate \
  --prompt "Ultra-realistic product shot of a ceramic mug on a wooden desk" \
  --images base.png ref1.png \
  --fragment fragments/photorealism.txt \
  --output output.png`,
    }

    cmd.Flags().StringSliceVar(&images, "images", []string{}, "Zero or more input image paths")
    cmd.Flags().StringVar(&prompt, "prompt", "", "Text prompt guiding the generation (required)")
    cmd.Flags().StringSliceVarP(&fragments, "fragment", "f", []string{}, "One or more text fragments to append to the prompt")
    cmd.Flags().StringVarP(&output, "output", "o", "output.png", "Path to save the generated PNG image")

    // Basic validation of files to provide helpful errors early
    cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
        for _, p := range images {
            if _, err := os.Stat(p); err != nil {
                return fmt.Errorf("image not found: %s", p)
            }
        }
        for _, f := range fragments {
            if _, err := os.Stat(f); err != nil {
                return fmt.Errorf("fragment not found: %s", f)
            }
        }
        if dir := filepath.Dir(output); dir != "." {
            if err := os.MkdirAll(dir, 0o755); err != nil {
                return fmt.Errorf("failed to create output dir %s: %w", dir, err)
            }
        }
        return nil
    }
    return cmd
}

func init() { rootCmd.AddCommand(newGenerateCmd()) }



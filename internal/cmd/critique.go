package cmd

import (
    "errors"
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var (
    critiqueImage string
    critiquePrompt string
    critiqueFragments []string
    critiquePrev string
)

func newCritiqueCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "critique",
        Short: "Critique an image for artifacts and prompt mismatches",
        Long:  "Run an expert forensic critique on the given image relative to the original prompt, with follow-on guidance if a previous critique is provided.",
        RunE: func(cmd *cobra.Command, args []string) error {
            if critiqueImage == "" { return errors.New("--image is required") }
            if _, err := os.Stat(critiqueImage); err != nil { return fmt.Errorf("image not found: %s", critiqueImage) }
            if critiquePrompt == "" { return errors.New("--prompt is required") }
            // Placeholder: call internal critique implementation
            fmt.Fprintln(cmd.OutOrStdout(), "[critique output placeholder]")
            return nil
        },
        Example: `nano-agent critique --image output.png --prompt "Studio portrait of..."`,
    }
    cmd.Flags().StringVar(&critiqueImage, "image", "", "Path to the input image (required)")
    cmd.Flags().StringVar(&critiquePrompt, "prompt", "", "Original prompt that guided the image (required)")
    cmd.Flags().StringSliceVarP(&critiqueFragments, "fragment", "f", []string{}, "Optional text fragments to include in critique prompt")
    cmd.Flags().StringVar(&critiquePrev, "prev-critique", "", "Optional previous critique to enable follow-on escalation")
    return cmd
}

func init() { rootCmd.AddCommand(newCritiqueCmd()) }



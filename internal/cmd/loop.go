package cmd

import (
    "errors"
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var (
    loopImages []string
    loopPrompt string
    loopFragments []string
    loopOutput string
    loopCount int
)

func newLoopCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "loop",
        Short: "Run a critique-improve loop for N iterations",
        Long:  "Generate or edit an image, then run iterative critique and re-generation for a given number of loops.",
        RunE: func(cmd *cobra.Command, args []string) error {
            if loopPrompt == "" { return errors.New("--prompt is required") }
            if loopOutput == "" { loopOutput = "output.png" }
            for _, p := range loopImages {
                if _, err := os.Stat(p); err != nil { return fmt.Errorf("image not found: %s", p) }
            }
            if loopCount < 0 { return errors.New("--critique-loops must be >= 0") }
            // Placeholder: call internal pipeline
            fmt.Fprintf(cmd.OutOrStdout(), "Running %d critique loops, output %s...\n", loopCount, loopOutput)
            return nil
        },
        Example: `nano-agent loop --prompt "Portrait of..." --images base.png -f fragments/realism.txt -o output.png -cl 3`,
    }
    cmd.Flags().StringSliceVar(&loopImages, "images", []string{}, "Zero or more input image paths")
    cmd.Flags().StringVar(&loopPrompt, "prompt", "", "Text prompt guiding the generation (required)")
    cmd.Flags().StringSliceVarP(&loopFragments, "fragment", "f", []string{}, "One or more text fragments to append to the prompt")
    cmd.Flags().StringVarP(&loopOutput, "output", "o", "output.png", "Path to save the generated PNG image")
    cmd.Flags().IntVarP(&loopCount, "critique-loops", "c", 0, "Number of critique-improve loops to run")
    return cmd
}

func init() { rootCmd.AddCommand(newLoopCmd()) }



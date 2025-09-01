package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rkirkendall/nano-agent/internal/ai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	images        []string
	fragments     []string
	prompt        string
	output        string
	critiqueLoops int

	rootCmd = &cobra.Command{
		Use:   "nano-agent [images...]",
		Short: "Nano Agent — image generation and critique CLI for Gemini",
		Long:  "Nano Agent is a cross-platform CLI that generates and iteratively improves images using Google's Gemini models with critique-improve loops.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Self-update check (best-effort, non-blocking)
			maybeSelfUpdate(cmd)
			// Treat positional args as image paths (Python parity)
			if len(args) > 0 {
				images = append(images, args...)
			}
			if strings.TrimSpace(prompt) == "" {
				return fmt.Errorf("--prompt is required")
			}
			if output == "" {
				output = "output.png"
			}
			if dir := filepath.Dir(output); dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("failed to create output dir %s: %w", dir, err)
				}
			}

			ctx := context.Background()
			model := viper.GetString("model")

			imgBytes, err := ai.GenerateImage(ctx, model, images, prompt, fragments)
			if err != nil {
				return err
			}
			if !strings.HasSuffix(strings.ToLower(output), ".png") {
				output += ".png"
			}
			if err := os.WriteFile(output, imgBytes, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Generated image saved at: %s\n", output)

			if critiqueLoops > 0 {
				baseOutputPath := output
				baseDir := filepath.Dir(baseOutputPath)
				baseName := strings.TrimSuffix(filepath.Base(baseOutputPath), filepath.Ext(baseOutputPath))
				outputsDir := filepath.Join(baseDir, "outputs")
				_ = os.MkdirAll(outputsDir, 0o755)

				currentImagePath := baseOutputPath
				var lastCritique string
				for i := 1; i <= critiqueLoops; i++ {
					fmt.Fprintf(cmd.OutOrStdout(), "\n=== Critique loop %d/%d ===\n", i, critiqueLoops)
					prev := ""
					if i > 1 {
						prev = lastCritique
					}
					critiqueText, err := ai.GenerateCritique(ctx, model, currentImagePath, prompt, fragments, prev)
					if err != nil {
						return fmt.Errorf("critique failed: %w", err)
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Critique feedback:")
					fmt.Fprintln(cmd.OutOrStdout(), critiqueText)

					improvementPrompt := "Apply the following critique to improve this image. Prioritize items tagged [CRITICAL — persisted] first, then [MAJOR], then [MINOR]. Use decisive, localized fixes and avoid regressions on items marked done. Then implement the 'Targeted actions to apply now' if present. Critique follows:\n\n" + critiqueText

					imgBytes, err := ai.GenerateImage(ctx, model, []string{currentImagePath}, improvementPrompt, fragments)
					if err != nil {
						return fmt.Errorf("improvement generation failed: %w", err)
					}
					if err := os.WriteFile(baseOutputPath, imgBytes, 0o644); err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Improved image saved at: %s\n", baseOutputPath)
					copyPath := filepath.Join(outputsDir, fmt.Sprintf("%s_improved_%d.png", baseName, i))
					src, err := os.Open(baseOutputPath)
					if err == nil {
						defer src.Close()
						if dst, err2 := os.Create(copyPath); err2 == nil {
							_, _ = io.Copy(dst, src)
							_ = dst.Close()
							fmt.Fprintf(cmd.OutOrStdout(), "Iteration copy saved at: %s\n", copyPath)
						}
					}
					lastCritique = critiqueText
				}
			}
			return nil
		},
		Example: `nano-agent --prompt "Portrait..." -o output.png base.png -f fragments/a.txt --critique-loops 3 (or: -cl 3)`,
	}
)

func normalizeArgs(argv []string) []string {
	if len(argv) == 0 {
		return argv
	}
	out := make([]string, 0, len(argv)+2)
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == "-cl" {
			out = append(out, "--critique-loops")
			if i+1 < len(argv) {
				out = append(out, argv[i+1])
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "-cl=") {
			out = append(out, "--critique-loops"+a[3:])
			continue
		}
		out = append(out, a)
	}
	return out
}

func Execute() {
	// Allow Python-style -cl flag
	os.Args = normalizeArgs(os.Args)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nano-agent.yaml)")
	rootCmd.PersistentFlags().String("model", "gemini-2.5-flash-image-preview", "Model to use for generation and critique")
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))

	rootCmd.Flags().StringSliceVar(&images, "images", []string{}, "Zero or more path(s) to input image files")
	rootCmd.Flags().StringSliceVarP(&fragments, "fragment", "f", []string{}, "One or more text files to append as reusable prompt fragments")
	rootCmd.Flags().StringVar(&prompt, "prompt", "", "Text prompt guiding the generation (required)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "output.png", "Path to save the generated PNG image")
	rootCmd.Flags().IntVar(&critiqueLoops, "critique-loops", 0, "Number of critique-improve loops to run (default: 0)")
}

func initConfig() {
	viper.AutomaticEnv()
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
			viper.SetConfigName(".nano-agent")
		}
	}
	_ = viper.ReadInConfig()
}

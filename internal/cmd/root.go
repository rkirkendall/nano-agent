package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rkirkendall/nano-agent/internal/ai"
	"github.com/rkirkendall/nano-agent/internal/generate"
	"github.com/rkirkendall/nano-agent/internal/version"
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
	versionFlag   bool
	transparent   bool

	rootCmd = &cobra.Command{
		Use:   "nano-agent [images...]",
		Short: "Nano Agent — image generation and critique CLI for Gemini",
		Long:  "Nano Agent is a cross-platform CLI that generates and iteratively improves images using Google's Gemini models with critique-improve loops.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// --version/-v: print version and exit
			if versionFlag {
				fmt.Fprintln(cmd.OutOrStdout(), version.Version)
				return nil
			}
			// Self-update check (best-effort, non-blocking)
			maybeSelfUpdate(cmd)
			// Treat positional args as image paths (Python parity)
			if len(args) > 0 {
				images = append(images, args...)
			}
			if strings.TrimSpace(prompt) == "" {
				return fmt.Errorf("--prompt is required")
			}
			// Validate that fragments are text files, not images
			for _, f := range fragments {
				ext := strings.ToLower(filepath.Ext(f))
				switch ext {
				case ".png", ".jpg", ".jpeg", ".webp", ".gif":
					return fmt.Errorf("--fragment expects text files; got image file: %s", f)
				}
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

			// If transparent requested, augment prompt with a white-background instruction
			effPrompt := prompt
			if transparent {
				hint := "Please render the subject with a solid white background around the subject; avoid interior transparency."
				if strings.TrimSpace(effPrompt) == "" {
					effPrompt = hint
				} else {
					effPrompt = effPrompt + "\n\n" + hint
				}
			}

			imgBytes, err := ai.GenerateImage(ctx, model, images, effPrompt, fragments)
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

			// Post-process to transparent background if requested
			if transparent {
				if err := makeBackgroundTransparent(output); err != nil {
					return fmt.Errorf("post-process transparency failed: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Applied transparent background post-process: %s\n", output)
			}

			if critiqueLoops > 0 {
				baseOutputPath := output
				baseDir := filepath.Dir(baseOutputPath)
				baseName := strings.TrimSuffix(filepath.Base(baseOutputPath), filepath.Ext(baseOutputPath))
				outputsDir := filepath.Join(baseDir, "outputs")
				_ = os.MkdirAll(outputsDir, 0o755)

				currentImagePath := baseOutputPath
				for i := 1; i <= critiqueLoops; i++ {
					fmt.Fprintf(cmd.OutOrStdout(), "\n=== Critique loop %d/%d ===\n", i, critiqueLoops)
					critiqueText, err := ai.GenerateCritique(ctx, model, currentImagePath, prompt, fragments, images)
					if err != nil {
						return fmt.Errorf("critique failed: %w", err)
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Critique feedback:")
					fmt.Fprintln(cmd.OutOrStdout(), critiqueText)

					improvementPrompt := generate.BuildImprovementPrompt(prompt, critiqueText)

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
	rootCmd.PersistentFlags().String("model", "gemini-2.5-flash-image-preview:free", "Model to use for generation and critique")
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))

	rootCmd.Flags().StringSliceVar(&images, "images", []string{}, "Zero or more path(s) to input image files")
	rootCmd.Flags().StringSliceVarP(&fragments, "fragment", "f", []string{}, "One or more text files to append as reusable prompt fragments")
	rootCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Text prompt guiding the generation (required)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "output.png", "Path to save the generated PNG image")
	rootCmd.Flags().IntVar(&critiqueLoops, "critique-loops", 0, "Number of critique-improve loops to run (default: 0)")
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.Flags().BoolVarP(&transparent, "transparent", "t", false, "Request white background and post-process to transparent background")
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

// makeBackgroundTransparent performs a flood-fill from the border to convert a white background
// to transparency using ImageMagick. Prefers IM7 `magick` and falls back to IM6 `convert`.
func makeBackgroundTransparent(path string) error {
	dir := filepath.Dir(path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	tmp := filepath.Join(dir, base+"__tmp_transparent.png")

	fuzz := strings.TrimSpace(os.Getenv("NANO_BG_FUZZ"))
	if fuzz == "" {
		fuzz = "6%"
	}
	// Args: add 1px white border, flood-fill from (1,1) to none, shave border
	// Use IM7-friendly draw primitive: 'color x,y floodfill'
	args := []string{path, "-colorspace", "sRGB", "-alpha", "set", "-bordercolor", "white", "-border", "1",
		"-fuzz", fuzz, "-fill", "none", "-draw", "color 1,1 floodfill", "-shave", "1x1", tmp}

	var cmd *exec.Cmd
	if _, err := exec.LookPath("magick"); err == nil {
		cmd = exec.Command("magick", args...)
	} else if _, err := exec.LookPath("convert"); err == nil {
		cmd = exec.Command("convert", args...)
	} else {
		return fmt.Errorf("ImageMagick not installed: install via Homebrew (brew install imagemagick) or apt (sudo apt-get install -y imagemagick)")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		b, rerr := os.ReadFile(tmp)
		if rerr != nil {
			_ = os.Remove(tmp)
			return err
		}
		if werr := os.WriteFile(path, b, 0o644); werr != nil {
			_ = os.Remove(tmp)
			return err
		}
		_ = os.Remove(tmp)
	}
	return nil
}

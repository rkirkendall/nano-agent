package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    rootCmd = &cobra.Command{
        Use:   "nano-agent",
        Short: "Nano Agent — image generation and critique CLI for Gemini",
        Long:  "Nano Agent is a cross-platform CLI that generates and iteratively improves images using Google's Gemini models with critique-improve loops.",
    }
)

func Execute() {
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



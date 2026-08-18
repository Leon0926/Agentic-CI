package commands

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Logger is the CLI-wide stderr logger, built in rootCmd's
// PersistentPreRun once flags are parsed (so --verbose is already known).
// Always os.Stderr: reviewd's diff-in/findings-out contract requires
// stdout to stay pure JSON, never log output.
var Logger *slog.Logger

var rootCmd = &cobra.Command{
	Use:           "reviewd",
	Short:         "reviewd is a AI code review tool for detecting defects and generating reports",
	Long:          "reviewd runs detector llm agents that can read files, grep the repo and later run tests in disposable work tree",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		lvl := slog.LevelInfo
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			lvl = slog.LevelDebug
		}
		Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	pf := rootCmd.PersistentFlags()
	pf.String("config", "", "config file default= .reviewd.yaml")
	pf.String("model", "claude-sonnet-4-6", "model to use for reviewd, default=claude-sonnet-4-6")
	pf.Int("max-iterations", 8, "max tool-call iterations per detector agent")
	pf.Float64("confidence-threshold", 0.5, "drop findings below this confidence")
	pf.Bool("verbose", false, "enable debug-level logging (stderr only)")

	_ = viper.BindPFlag("model", pf.Lookup("model"))
	_ = viper.BindPFlag("max-iterations", pf.Lookup("max-iterations"))
	_ = viper.BindPFlag("confidence-threshold", pf.Lookup("confidence-threshold"))
}

func initConfig() {

	if cfg, _ := rootCmd.PersistentFlags().GetString("config"); cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.SetConfigName(".reviewd")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}
	viper.SetEnvPrefix("REVIEWD")
	viper.AutomaticEnv()
	// Missing config file is fine as flags/env/defaults cover everything
	_ = viper.ReadInConfig()
}

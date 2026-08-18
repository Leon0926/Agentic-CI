package commands

import (
	"errors"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Logger is the CLI-wide stderr logger, built in initConfig (via
// cobra.OnInitialize) once flags are parsed, so --verbose is already
// known and it's available in time to warn about config-load problems.
// Always os.Stderr: reviewd's diff-in/findings-out contract requires
// stdout to stay pure JSON, never log output.
var Logger *slog.Logger

var rootCmd = &cobra.Command{
	Use:           "reviewd",
	Short:         "reviewd is a AI code review tool for detecting defects and generating reports",
	Long:          "reviewd runs detector llm agents that can read files, grep the repo and later run tests in disposable work tree",
	SilenceUsage:  true,
	SilenceErrors: true,
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
	lvl := slog.LevelInfo
	if verbose, _ := rootCmd.PersistentFlags().GetBool("verbose"); verbose {
		lvl = slog.LevelDebug
	}
	Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))

	if cfg, _ := rootCmd.PersistentFlags().GetString("config"); cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.SetConfigName(".reviewd")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}
	viper.SetEnvPrefix("REVIEWD")
	viper.AutomaticEnv()

	// A missing/broken config file is fine — flags/env/defaults cover
	// everything — but it's exactly what makes a detector silently
	// disable itself (see: secrets_llm.enabled reading as false), so
	// warn loudly instead of swallowing the error.
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			wd, _ := os.Getwd()
			Logger.Warn("no .reviewd.yaml found; using flags/env/defaults only", "cwd", wd)
		} else {
			Logger.Warn("config file found but failed to load; using flags/env/defaults only", "error", err)
		}
	}
}

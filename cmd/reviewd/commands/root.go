package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "reviewd",
	Short: "reviewd is a AI code review tool for detecting defects and generating reports",
	Long:  "reviewd runs detector llm agents that can read files, grep the repo and later run tests in disposable work tree",
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

	_ = viper.BindPFlag("config", pf.Lookup("config"))
	_ = viper.BindPFlag("model", pf.Lookup("model"))
	_ = viper.BindPFlag("max-iterations", pf.Lookup("max-iterations"))
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

/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	telegrambot "gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/shillgptbot"
)

const (
	cmdEnvPrefix = "shill_bot"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "shill-gpt-bot",
	Short: "Telegram Shill Bot",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.shill-bot.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	telegrambot.InitConfig(cfgFile)
}

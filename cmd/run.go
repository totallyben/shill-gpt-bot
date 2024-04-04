/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/shillgptbot"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the bot",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		runCmdValidate(cmd)

		sb := shillgptbot.NewShillGPTBot()
		sb.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// read in env vars
	viper.SetEnvPrefix(cmdEnvPrefix)
	viper.AutomaticEnv()
}

func runCmdValidate(cmd *cobra.Command) {

}

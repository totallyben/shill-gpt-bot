package cmd

import (
	"github.com/labstack/echo/v4"
	"gitlab.totallydev.com/gritzb/shill-gpt-bot/pkg/api"

	"github.com/spf13/cobra"
)

var port *int

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API service",
	Long:  `The server runs on port 8080 by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		api := api.NewApi(*port)
		api.Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	port = serveCmd.Flags().Int("port", 8080, "Port number the API runs on")
}

func healthCheck(c echo.Context) error {
	return api.ReturnSuccessMessage(c, "We're alive!")
}

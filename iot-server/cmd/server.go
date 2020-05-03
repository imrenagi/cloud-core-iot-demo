package cmd

import (
	"net/http"

	"github.com/imrenagi/iot-demo-server/web"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start alpha server",
	Long:  "Start alpha server",
	Run: func(cmd *cobra.Command, args []string) {

		log.Warn().Msg("Starting server")
		s := web.NewServer()
		if err := http.ListenAndServe(":8080", s.GetHandler()); err != nil {
			log.Fatal().Msgf("Server can't run. Got: `%v`", err)
		}
	},
}

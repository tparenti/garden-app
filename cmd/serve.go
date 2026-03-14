package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alexr/garden-app/internal/web"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI",
	RunE:  runServe,
}

var servePort int

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
}

func runServe(cmd *cobra.Command, _ []string) error {
	ac := getAppContext(cmd)
	srv := web.NewServer(&web.AppContext{
		Store:    ac.Store,
		FrostSvc: ac.FrostSvc,
		Calc:     ac.Calc,
	}, servePort)
	fmt.Printf("Garden UI → http://localhost:%d\n", servePort)
	return srv.ListenAndServe()
}

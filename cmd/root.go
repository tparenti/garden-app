package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/alexr/garden-app/internal/planting"
	"github.com/alexr/garden-app/internal/store"
)

// AppContext holds shared dependencies injected into subcommands.
type AppContext struct {
	Store    store.Store
	FrostSvc *planting.FrostDateService
	Calc     *planting.Calculator
}

type contextKey struct{}

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "garden",
	Short: "Garden organizer — seed inventory, planting schedule, and frost-date planning",
	Long: `garden helps you track your seed inventory, plan planting schedules,
and calculates optimal planting windows based on your local frost dates.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultDB := filepath.Join(userHomeDir(), ".garden", "garden.db")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to the garden database")

	cobra.OnInitialize(initApp)
}

// initApp is registered via cobra.OnInitialize and runs before every subcommand.
// Note: do NOT guard this with rootCmd.Args == nil — that field is the cobra
// PositionalArgs validator function, not the actual CLI arguments, and is nil
// whenever no validator is set. Such a guard causes initApp to return early on
// every invocation, leaving the app context unset and crashing subcommands.
func initApp() {
	frostSvc, err := planting.NewFrostDateService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading frost data: %v\n", err)
		os.Exit(1)
	}

	st, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		os.Exit(1)
	}

	appCtx := &AppContext{
		Store:    st,
		FrostSvc: frostSvc,
		Calc:     planting.NewCalculator(frostSvc),
	}

	// Inject into cobra's context
	ctx := context.WithValue(context.Background(), contextKey{}, appCtx)
	rootCmd.SetContext(ctx)
}

// getAppContext retrieves the AppContext from a cobra command's context.
func getAppContext(cmd *cobra.Command) *AppContext {
	ac, ok := cmd.Root().Context().Value(contextKey{}).(*AppContext)
	if !ok || ac == nil {
		fmt.Fprintln(os.Stderr, "internal error: app context not initialized")
		os.Exit(1)
	}
	return ac
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

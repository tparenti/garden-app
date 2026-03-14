package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/alexr/garden-app/internal/models"
)

var seedsCmd = &cobra.Command{
	Use:   "seeds",
	Short: "Manage your seed inventory",
}

var seedsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add seeds to your inventory",
	RunE:  runSeedsAdd,
}

var seedsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all seeds in your inventory",
	RunE:  runSeedsList,
}

var seedsRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a seed from your inventory",
	Args:  cobra.ExactArgs(1),
	RunE:  runSeedsRemove,
}

var seedsLinkCmd = &cobra.Command{
	Use:   "link <seed-id> <spec-id>",
	Short: "Link a seed to a plant spec",
	Args:  cobra.ExactArgs(2),
	RunE:  runSeedsLink,
}

var (
	seedName    string
	seedVariety string
	seedQty     int
	seedUnit    string
	seedNotes   string
	seedSpecID  int64
)

func init() {
	rootCmd.AddCommand(seedsCmd)
	seedsCmd.AddCommand(seedsAddCmd, seedsListCmd, seedsRemoveCmd, seedsLinkCmd)

	seedsAddCmd.Flags().StringVarP(&seedName, "name", "n", "", "plant name (required)")
	seedsAddCmd.Flags().StringVarP(&seedVariety, "variety", "v", "", "variety")
	seedsAddCmd.Flags().IntVarP(&seedQty, "qty", "q", 1, "quantity")
	seedsAddCmd.Flags().StringVarP(&seedUnit, "unit", "u", "packets", "unit (packets, grams, seeds)")
	seedsAddCmd.Flags().StringVar(&seedNotes, "notes", "", "notes")
	seedsAddCmd.Flags().Int64Var(&seedSpecID, "spec-id", 0, "link to a plant spec by ID")
	_ = seedsAddCmd.MarkFlagRequired("name")
}

func runSeedsAdd(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	now := time.Now()
	s := &models.Seed{
		Name:        seedName,
		Variety:     seedVariety,
		Quantity:    seedQty,
		Unit:        seedUnit,
		Notes:       seedNotes,
		PurchasedAt: &now,
	}
	if seedSpecID > 0 {
		s.PlantSpecID = &seedSpecID
	}
	id, err := ac.Store.AddSeed(cmd.Context(), s)
	if err != nil {
		return err
	}
	fmt.Printf("Added seed #%d: %s %s\n", id, seedName, seedVariety)
	return nil
}

func runSeedsList(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	seeds, err := ac.Store.ListSeeds(cmd.Context())
	if err != nil {
		return err
	}
	if len(seeds) == 0 {
		fmt.Println("No seeds in inventory. Use 'garden seeds add' to add some.")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Variety", "Qty", "Unit", "Spec ID", "Notes"})
	table.SetBorder(true)
	table.SetRowLine(false)

	for _, s := range seeds {
		specID := ""
		if s.PlantSpecID != nil {
			specID = fmt.Sprintf("%d", *s.PlantSpecID)
		}
		notes := s.Notes
		if len(notes) > 30 {
			notes = notes[:27] + "..."
		}
		table.Append([]string{
			fmt.Sprintf("%d", s.ID),
			s.Name,
			s.Variety,
			fmt.Sprintf("%d", s.Quantity),
			s.Unit,
			specID,
			notes,
		})
	}
	table.Render()
	return nil
}

func runSeedsRemove(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID %q", args[0])
	}
	if err := ac.Store.RemoveSeed(cmd.Context(), id); err != nil {
		return err
	}
	fmt.Printf("Removed seed #%d\n", id)
	return nil
}

func runSeedsLink(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	seedID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid seed ID %q", args[0])
	}
	specID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid spec ID %q", args[1])
	}

	seed, err := ac.Store.GetSeed(cmd.Context(), seedID)
	if err != nil {
		return err
	}
	seed.PlantSpecID = &specID
	if err := ac.Store.UpdateSeed(cmd.Context(), seed); err != nil {
		return err
	}
	fmt.Printf("Linked seed #%d to spec #%d\n", seedID, specID)
	return nil
}

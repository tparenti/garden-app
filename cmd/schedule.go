package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/alexr/garden-app/internal/models"
	"github.com/alexr/garden-app/internal/planting"
	"github.com/alexr/garden-app/internal/store"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage your planting schedule",
}

var scheduleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a planting entry to your schedule",
	RunE:  runScheduleAdd,
}

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled planting entries",
	RunE:  runScheduleList,
}

var scheduleSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest planting dates for a plant based on your locale",
	RunE:  runScheduleSuggest,
}

var scheduleDoneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Mark a planting entry as completed",
	Args:  cobra.ExactArgs(1),
	RunE:  runScheduleDone,
}

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a planting entry",
	Args:  cobra.ExactArgs(1),
	RunE:  runScheduleRemove,
}

var (
	entryPlant    string
	entryType     string
	entryDate     string
	entryLocation string
	entryQty      int
	entrySeedID   int64
	entrySpecID   int64
	entryNotes    string

	listFrom  string
	listTo    string
	listPlant string
	listType  string

	suggestPlant  string
	suggestSpecID int64
	suggestZip    string
	suggestYear   int

	doneDate string
)

func init() {
	rootCmd.AddCommand(scheduleCmd)
	scheduleCmd.AddCommand(scheduleAddCmd, scheduleListCmd, scheduleSuggestCmd, scheduleDoneCmd, scheduleRemoveCmd)

	scheduleAddCmd.Flags().StringVarP(&entryPlant, "plant", "p", "", "plant name (required)")
	scheduleAddCmd.Flags().StringVarP(&entryType, "type", "t", "direct_sow", "type: indoor_start, transplant, direct_sow")
	scheduleAddCmd.Flags().StringVarP(&entryDate, "date", "d", "", "planned date (YYYY-MM-DD, default: today)")
	scheduleAddCmd.Flags().StringVarP(&entryLocation, "location", "l", "", "garden location or bed name")
	scheduleAddCmd.Flags().IntVarP(&entryQty, "qty", "q", 0, "quantity planted")
	scheduleAddCmd.Flags().Int64Var(&entrySeedID, "seed-id", 0, "link to a seed in inventory")
	scheduleAddCmd.Flags().Int64Var(&entrySpecID, "spec-id", 0, "link to a plant spec")
	scheduleAddCmd.Flags().StringVar(&entryNotes, "notes", "", "notes")
	_ = scheduleAddCmd.MarkFlagRequired("plant")

	scheduleListCmd.Flags().StringVar(&listFrom, "from", "", "show entries from date (YYYY-MM-DD)")
	scheduleListCmd.Flags().StringVar(&listTo, "to", "", "show entries to date (YYYY-MM-DD)")
	scheduleListCmd.Flags().StringVar(&listPlant, "plant", "", "filter by plant name")
	scheduleListCmd.Flags().StringVar(&listType, "type", "", "filter by type")

	scheduleSuggestCmd.Flags().StringVarP(&suggestPlant, "plant", "p", "", "plant name to search for")
	scheduleSuggestCmd.Flags().Int64Var(&suggestSpecID, "spec-id", 0, "plant spec ID")
	scheduleSuggestCmd.Flags().StringVar(&suggestZip, "zip", "", "zip code (overrides stored locale)")
	scheduleSuggestCmd.Flags().IntVar(&suggestYear, "year", time.Now().Year(), "year for planting window calculation")

	scheduleDoneCmd.Flags().StringVarP(&doneDate, "date", "d", "", "actual date completed (YYYY-MM-DD, default: today)")
}

func runScheduleAdd(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)

	planned := time.Now()
	if entryDate != "" {
		t, err := time.ParseInLocation("2006-01-02", entryDate, time.Local)
		if err != nil {
			return fmt.Errorf("invalid date %q, use YYYY-MM-DD", entryDate)
		}
		planned = t
	}

	e := &models.PlantingEntry{
		PlantName:       entryPlant,
		PlantingType:    entryType,
		PlannedDate:     planned,
		Location:        entryLocation,
		QuantityPlanted: entryQty,
		Notes:           entryNotes,
	}
	if entrySeedID > 0 {
		e.SeedID = &entrySeedID
	}
	if entrySpecID > 0 {
		e.PlantSpecID = &entrySpecID
	}

	id, err := ac.Store.AddPlantingEntry(cmd.Context(), e)
	if err != nil {
		return err
	}
	fmt.Printf("Scheduled #%d: %s (%s) on %s\n", id, entryPlant, entryType, planned.Format("2006-01-02"))
	return nil
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)

	filter := store.PlantingFilter{
		PlantName: listPlant,
		Type:      listType,
	}
	if listFrom != "" {
		t, err := time.ParseInLocation("2006-01-02", listFrom, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --from date: %w", err)
		}
		filter.FromDate = &t
	}
	if listTo != "" {
		t, err := time.ParseInLocation("2006-01-02", listTo, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --to date: %w", err)
		}
		filter.ToDate = &t
	}

	entries, err := ac.Store.ListPlantingEntries(cmd.Context(), filter)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No scheduled entries found. Use 'garden schedule add' to add some.")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Plant", "Type", "Planned", "Done", "Location", "Qty"})
	table.SetBorder(true)

	for _, e := range entries {
		done := ""
		if e.ActualDate != nil {
			done = e.ActualDate.Format("2006-01-02")
		}
		table.Append([]string{
			fmt.Sprintf("%d", e.ID),
			e.PlantName,
			e.PlantingType,
			e.PlannedDate.Format("2006-01-02"),
			done,
			e.Location,
			fmt.Sprintf("%d", e.QuantityPlanted),
		})
	}
	table.Render()
	return nil
}

func runScheduleSuggest(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)

	// Resolve zip from flag or stored locale
	zip := suggestZip
	state := ""
	if zip == "" {
		stored, err := ac.Store.GetConfig(cmd.Context(), "zip")
		if err == nil {
			zip = stored
		} else {
			stored, err = ac.Store.GetConfig(cmd.Context(), "state")
			if err == nil {
				state = stored
			}
		}
	}
	if zip == "" && state == "" {
		return fmt.Errorf("no locale set. Use --zip or run 'garden locale set --zip <zipcode>'")
	}

	// Resolve plant spec
	var spec *models.PlantSpec

	if suggestSpecID > 0 {
		s, err := ac.Store.GetPlantSpec(cmd.Context(), suggestSpecID)
		if err != nil {
			return err
		}
		spec = s
	} else if suggestPlant != "" {
		specs, err := ac.Store.SearchPlantSpecs(cmd.Context(), suggestPlant)
		if err != nil {
			return err
		}
		if len(specs) == 0 {
			return fmt.Errorf("no plant spec found for %q — use 'garden plants search' to browse", suggestPlant)
		}
		if len(specs) > 1 {
			fmt.Printf("Multiple specs found for %q. Showing suggestions for each:\n\n", suggestPlant)
			for i := range specs {
				win, err := ac.Calc.CalculateWindow(&specs[i], zip, state, suggestYear)
				if err != nil {
					fmt.Printf("  Spec #%d (%s %s): %v\n", specs[i].ID, specs[i].Name, specs[i].Variety, err)
					continue
				}
				fmt.Printf("Spec #%d — Variety: %s\n", specs[i].ID, specs[i].Variety)
				fmt.Println(planting.FormatWindow(win))
			}
			return nil
		}
		spec = &specs[0]
	} else {
		return fmt.Errorf("specify --plant or --spec-id")
	}

	win, err := ac.Calc.CalculateWindow(spec, zip, state, suggestYear)
	if err != nil {
		return err
	}
	fmt.Println(planting.FormatWindow(win))
	return nil
}

func runScheduleDone(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID %q", args[0])
	}

	e, err := ac.Store.GetPlantingEntry(cmd.Context(), id)
	if err != nil {
		return err
	}

	t := time.Now()
	if doneDate != "" {
		t, err = time.ParseInLocation("2006-01-02", doneDate, time.Local)
		if err != nil {
			return fmt.Errorf("invalid date %q, use YYYY-MM-DD", doneDate)
		}
	}
	e.ActualDate = &t

	if err := ac.Store.UpdatePlantingEntry(cmd.Context(), e); err != nil {
		return err
	}
	fmt.Printf("Marked #%d (%s) as done on %s\n", id, e.PlantName, t.Format("2006-01-02"))
	return nil
}

func runScheduleRemove(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID %q", args[0])
	}
	if err := ac.Store.RemovePlantingEntry(cmd.Context(), id); err != nil {
		return err
	}
	fmt.Printf("Removed schedule entry #%d\n", id)
	return nil
}

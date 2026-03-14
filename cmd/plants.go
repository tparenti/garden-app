package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var plantsCmd = &cobra.Command{
	Use:   "plants",
	Short: "Browse the plant spec library",
}

var plantsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plant specs",
	RunE:  runPlantsList,
}

var plantsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show detailed info for a plant spec",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlantsShow,
}

var plantsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search plant specs by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlantsSearch,
}

var sunFilter string

func init() {
	rootCmd.AddCommand(plantsCmd)
	plantsCmd.AddCommand(plantsListCmd, plantsShowCmd, plantsSearchCmd)

	plantsListCmd.Flags().StringVar(&sunFilter, "sun", "", "filter by sun requirement (full, partial, shade)")
}

func runPlantsList(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	specs, err := ac.Store.ListPlantSpecs(cmd.Context())
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Variety", "Maturity", "Sun", "Water", "Start Indoors", "Direct Sow"})
	table.SetBorder(true)

	for _, s := range specs {
		if sunFilter != "" && !strings.EqualFold(s.SunRequirement, sunFilter) {
			continue
		}
		startIndoors := "No"
		if s.StartIndoors {
			startIndoors = fmt.Sprintf("Yes (%dw before)", s.WeeksBeforeFrost)
		}
		directSow := "No"
		if s.DirectSow {
			sign := "after"
			weeks := s.WeeksAfterFrost
			if weeks < 0 {
				sign = "before"
				weeks = -weeks
			}
			directSow = fmt.Sprintf("Yes (%dw %s)", weeks, sign)
		}
		table.Append([]string{
			fmt.Sprintf("%d", s.ID),
			s.Name,
			s.Variety,
			fmt.Sprintf("%d days", s.DaysToMaturity),
			s.SunRequirement,
			s.WaterRequirement,
			startIndoors,
			directSow,
		})
	}
	table.Render()
	return nil
}

func runPlantsShow(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID %q", args[0])
	}
	s, err := ac.Store.GetPlantSpec(cmd.Context(), id)
	if err != nil {
		return err
	}

	fmt.Printf("\n  Plant Spec #%d\n", s.ID)
	fmt.Printf("  %-22s %s\n", "Name:", s.Name)
	fmt.Printf("  %-22s %s\n", "Variety:", s.Variety)
	fmt.Printf("  %-22s %d days\n", "Days to Germinate:", s.DaysToGermination)
	fmt.Printf("  %-22s %d days\n", "Days to Maturity:", s.DaysToMaturity)
	fmt.Printf("  %-22s %.1f inches\n", "Spacing:", s.SpacingInches)
	fmt.Printf("  %-22s %.2f inches\n", "Planting Depth:", s.DepthInches)
	fmt.Printf("  %-22s %s\n", "Sun Requirement:", s.SunRequirement)
	fmt.Printf("  %-22s %s\n", "Water Requirement:", s.WaterRequirement)
	if s.StartIndoors {
		fmt.Printf("  %-22s %d weeks before last frost\n", "Start Indoors:", s.WeeksBeforeFrost)
	}
	if s.DirectSow {
		if s.WeeksAfterFrost >= 0 {
			fmt.Printf("  %-22s %d weeks after last frost\n", "Direct Sow:", s.WeeksAfterFrost)
		} else {
			fmt.Printf("  %-22s %d weeks before last frost\n", "Direct Sow:", -s.WeeksAfterFrost)
		}
	}
	fmt.Printf("  %-22s %s – %s\n", "Hardiness Zones:", s.HardinessZoneMin, s.HardinessZoneMax)
	if s.Notes != "" {
		fmt.Printf("\n  Notes: %s\n", s.Notes)
	}
	fmt.Println()
	return nil
}

func runPlantsSearch(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)
	specs, err := ac.Store.SearchPlantSpecs(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if len(specs) == 0 {
		fmt.Printf("No plant specs found matching %q\n", args[0])
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Variety", "Maturity", "Sun"})
	table.SetBorder(true)

	for _, s := range specs {
		table.Append([]string{
			fmt.Sprintf("%d", s.ID),
			s.Name,
			s.Variety,
			fmt.Sprintf("%d days", s.DaysToMaturity),
			s.SunRequirement,
		})
	}
	table.Render()
	return nil
}

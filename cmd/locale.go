package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var localeCmd = &cobra.Command{
	Use:   "locale",
	Short: "Set or view your location for frost date calculations",
}

var localeSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set your locale (zip code or state)",
	RunE:  runLocaleSet,
}

var localeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show your current locale and frost dates",
	RunE:  runLocaleShow,
}

var (
	localeZip   string
	localeState string
)

func init() {
	rootCmd.AddCommand(localeCmd)
	localeCmd.AddCommand(localeSetCmd, localeShowCmd)

	localeSetCmd.Flags().StringVar(&localeZip, "zip", "", "5-digit US zip code")
	localeSetCmd.Flags().StringVar(&localeState, "state", "", "2-letter US state abbreviation (e.g. CO, WI)")
}

func runLocaleSet(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)

	if localeZip == "" && localeState == "" {
		return fmt.Errorf("specify --zip or --state")
	}

	if localeZip != "" {
		// Validate the zip resolves to frost data
		fd, err := ac.FrostSvc.LookupByZip(localeZip)
		if err != nil {
			return fmt.Errorf("zip %q not found in frost database: %w", localeZip, err)
		}
		if err := ac.Store.SetConfig(cmd.Context(), "zip", localeZip); err != nil {
			return err
		}
		// Clear state if zip is set
		_ = ac.Store.SetConfig(cmd.Context(), "state", "")
		fmt.Printf("Locale set to zip %s (%s, %s)\n", localeZip, fd.City, fd.State)
		fmt.Printf("Last spring frost: ~%s | First fall frost: ~%s\n",
			formatMMDD(fd.LastFrostMMDD), formatMMDD(fd.FirstFrostMMDD))
	} else {
		fd, err := ac.FrostSvc.LookupByState(localeState)
		if err != nil {
			return fmt.Errorf("state %q not found: %w", localeState, err)
		}
		if err := ac.Store.SetConfig(cmd.Context(), "state", localeState); err != nil {
			return err
		}
		_ = ac.Store.SetConfig(cmd.Context(), "zip", "")
		fmt.Printf("Locale set to state %s (representative: %s)\n", localeState, fd.City)
		fmt.Printf("Last spring frost: ~%s | First fall frost: ~%s\n",
			formatMMDD(fd.LastFrostMMDD), formatMMDD(fd.FirstFrostMMDD))
	}
	return nil
}

func runLocaleShow(cmd *cobra.Command, args []string) error {
	ac := getAppContext(cmd)

	zip, _ := ac.Store.GetConfig(cmd.Context(), "zip")
	state, _ := ac.Store.GetConfig(cmd.Context(), "state")

	if zip == "" && state == "" {
		fmt.Println("No locale set. Use 'garden locale set --zip <zipcode>' to configure.")
		return nil
	}

	if zip != "" {
		result, err := ac.FrostSvc.LookupByZip(zip)
		if err != nil {
			return err
		}
		fmt.Printf("Locale:             zip %s\n", zip)
		fmt.Printf("City/State:         %s, %s\n", result.City, result.State)
		fmt.Printf("Last spring frost:  ~%s\n", formatMMDD(result.LastFrostMMDD))
		fmt.Printf("First fall frost:   ~%s\n", formatMMDD(result.FirstFrostMMDD))
	} else {
		result, err := ac.FrostSvc.LookupByState(state)
		if err != nil {
			return err
		}
		fmt.Printf("Locale:             state %s\n", state)
		fmt.Printf("Representative:     %s, %s\n", result.City, result.State)
		fmt.Printf("Last spring frost:  ~%s\n", formatMMDD(result.LastFrostMMDD))
		fmt.Printf("First fall frost:   ~%s\n", formatMMDD(result.FirstFrostMMDD))
	}
	return nil
}

// formatMMDD converts "0415" to "April 15" style display.
func formatMMDD(mmdd string) string {
	if len(mmdd) != 4 {
		return mmdd
	}
	months := []string{
		"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	m := int(mmdd[0]-'0')*10 + int(mmdd[1]-'0')
	d := int(mmdd[2]-'0')*10 + int(mmdd[3]-'0')
	if m < 1 || m > 12 || d < 1 || d > 31 {
		if mmdd == "0000" {
			return "No frost (tropical)"
		}
		return mmdd
	}
	return fmt.Sprintf("%s %d", months[m], d)
}

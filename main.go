package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	pinCode, state, district, email, password, date, vaccine, fee string

	age, interval, dose, quantity int

	rootCmd = &cobra.Command{
		Use:   "covaccine-notifier [FLAGS]",
		Short: "CoWIN Vaccine availability notifier India",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(args)
		},
	}
)

const (
	pinCodeEnv        = "PIN_CODE"
	stateNameEnv      = "STATE_NAME"
	districtNameEnv   = "DISTRICT_NAME"
	ageEnv            = "AGE"
	emailIDEnv        = "EMAIL_ID"
	emailPasswordEnv  = "EMAIL_PASSOWORD"
	searchIntervalEnv = "SEARCH_INTERVAL"
	vaccineEnv        = "VACCINE"
	feeEnv            = "FEE"
	dateEnv           = "DATE"
	doseEnv           = "DOSE"
	quantityEnv       = "QUANTITY"

	defaultSearchInterval  = 60
	defaultDose            = 1
	defaultMinimumQuantity = 1

	covishield = "covishield"
	covaxin    = "covaxin"

	free = "free"
	paid = "paid"
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&pinCode, "pincode", "c", os.Getenv(pinCodeEnv), "Search by pin code")
	rootCmd.PersistentFlags().StringVarP(&state, "state", "s", os.Getenv(stateNameEnv), "Search by state name")
	rootCmd.PersistentFlags().StringVarP(&district, "district", "d", os.Getenv(districtNameEnv), "Search by district name")
	rootCmd.PersistentFlags().IntVarP(&age, "age", "a", getIntEnv(ageEnv), "Search appointment for age")
	rootCmd.PersistentFlags().StringVarP(&email, "email", "e", os.Getenv(emailIDEnv), "Email address to send notifications")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", os.Getenv(emailPasswordEnv), "Email ID password for auth")
	rootCmd.PersistentFlags().IntVarP(&interval, "interval", "i", getIntEnv(searchIntervalEnv), fmt.Sprintf("Interval to repeat the search. Default: (%v) second", defaultSearchInterval))
	rootCmd.PersistentFlags().StringVarP(&vaccine, "vaccine", "v", os.Getenv(vaccineEnv), "Vaccine preferences - covishield (or) covaxin. Default: No preference")
	rootCmd.PersistentFlags().StringVarP(&fee, "fee", "f", os.Getenv(feeEnv), "Fee preferences - free (or) paid. Default: No preference")
	rootCmd.PersistentFlags().StringVar(&date, "date", os.Getenv(dateEnv), "Appointment date to check from (DD-MM-YYYY). Default: Today")
	rootCmd.PersistentFlags().IntVar(&dose, "dose", getIntEnv(doseEnv), "Dose number - 1 (or) 2. Default: 1")
	rootCmd.PersistentFlags().IntVarP(&quantity, "quantity", "q", getIntEnv(quantityEnv), "Minimum number of vaccines - 1 (to) 5. Default: 1")
}

// Execute executes the main command
func Execute() error {
	return rootCmd.Execute()
}

func checkFlags() error {
	if len(pinCode) == 0 &&
		len(state) == 0 &&
		len(district) == 0 {
		return errors.New("Please pass one of the pinCode or state & district name combination options")
	}
	if len(pinCode) == 0 && (len(state) == 0 || len(district) == 0) {
		return errors.New("Missing state or district name option")
	}
	if age == 0 {
		return errors.New("Missing age option")
	}
	if len(email) == 0 || len(password) == 0 {
		return errors.New("Missing email creds")
	}
	if interval == 0 {
		interval = defaultSearchInterval
	}
	if !(vaccine == "" || vaccine == covishield || vaccine == covaxin) {
		return errors.New("Invalid vaccine, please use covaxin or covishield")
	}
	if !(fee == "" || fee == free || fee == paid) {
		return errors.New("Invalid fee preference, please use free or paid")
	}
	if len(date) != 0 {
		if _, err := time.Parse("02-01-2006", date); err != nil {
			errors.New("Appointment date must be of the format DD-MM-YYYY")
		}
	} else {
		date = timeNow()
	}
	if dose == 0 {
		dose = defaultDose
	}
	if !(dose == 1 || dose == 2) {
		return errors.New("Invalid dose option, please use 1 or 2")
	}
	if quantity == 0 {
		quantity = defaultMinimumQuantity
	}
	if quantity < defaultMinimumQuantity || quantity > 5 {
		return errors.New("Invalid quantity option, please enter value between 0-5")
	}
	return nil
}

func main() {
	Execute()
}

func getIntEnv(envVar string) int {
	v := os.Getenv(envVar)
	if len(v) == 0 {
		return 0
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Fatal(err)
	}
	return i
}

func Run(args []string) error {
	if err := checkFlags(); err != nil {
		return err
	}
	if err := checkSlots(); err != nil {
		return err
	}
	ticker := time.NewTicker(time.Second * time.Duration(interval))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := checkSlots(); err != nil {
				return err
			}
		}
	}
}

func checkSlots() error {
	// Search for slots
	if len(pinCode) != 0 {
		return searchByPincode(pinCode)
	}
	return searchByStateDistrict(age, state, district)
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
)

// https://apisetu.gov.in/public/api/cowin
const (
	baseURL                     = "https://cdn-api.co-vin.in/api"
	calendarByPinURLFormat      = "/v2/appointment/sessions/calendarByPin?pincode=%s&date=%s"
	calendarByDistrictURLFormat = "/v2/appointment/sessions/calendarByDistrict?district_id=%d&date=%s"
	listStatesURLFormat         = "/v2/admin/location/states"
	listDistrictsURLFormat      = "/v2/admin/location/districts/%d"
)

var (
	stateID, districtID int
)

type StateList struct {
	States []struct {
		StateID    int    `json:"state_id"`
		StateName  string `json:"state_name"`
		StateNameL string `json:"state_name_l"`
	} `json:"states"`
	TTL int `json:"ttl"`
}

type DistrictList struct {
	Districts []struct {
		StateID       int    `json:"state_id"`
		DistrictID    int    `json:"district_id"`
		DistrictName  string `json:"district_name"`
		DistrictNameL string `json:"district_name_l"`
	} `json:"districts"`
	TTL int `json:"ttl"`
}

type Appointments struct {
	Centers []struct {
		CenterID      int     `json:"center_id"`
		Name          string  `json:"name"`
		NameL         string  `json:"name_l"`
		StateName     string  `json:"state_name"`
		StateNameL    string  `json:"state_name_l"`
		DistrictName  string  `json:"district_name"`
		DistrictNameL string  `json:"district_name_l"`
		BlockName     string  `json:"block_name"`
		BlockNameL    string  `json:"block_name_l"`
		Pincode       int     `json:"pincode"`
		Lat           float64 `json:"lat"`
		Long          float64 `json:"long"`
		From          string  `json:"from"`
		To            string  `json:"to"`
		FeeType       string  `json:"fee_type"`
		VaccineFees   []struct {
			Vaccine string `json:"vaccine"`
			Fee     string `json:"fee"`
		} `json:"vaccine_fees"`
		Sessions []Sessions `json:"sessions"`
	} `json:"centers"`
}

type Sessions struct {
	SessionID              string   `json:"session_id"`
	Date                   string   `json:"date"`
	AvailableCapacity      int      `json:"available_capacity"`
	MinAgeLimit            int      `json:"min_age_limit"`
	Vaccine                string   `json:"vaccine"`
	Slots                  []string `json:"slots"`
	AvailableCapacityDose1 int      `json:"available_capacity_dose1"`
	AvailableCapacityDose2 int      `json:"available_capacity_dose2"`
}

func timeNow() string {
	return time.Now().Format("02-01-2006")
}

func queryServer(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "hi_IN")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36 Edg/90.0.818.51")

	log.Print("Querying endpoint: ", baseURL+path)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Print("Response: ", string(bodyBytes))

	// If API response is anything but "OK", do not fail in that case
	if resp.StatusCode != http.StatusOK {
		log.Printf("Request failed with statusCode: %d", resp.StatusCode)
		return nil, nil
	}

	return bodyBytes, nil
}

func searchByPincode(pinCode string) error {
	response, err := queryServer(fmt.Sprintf(calendarByPinURLFormat, pinCode, date))
	if err != nil {
		return errors.Wrap(err, "Failed to fetch appointment sessions")
	}
	return getAvailableSessions(response, age)
}

func getStateIDByName(state string) (int, error) {
	response, err := queryServer(listStatesURLFormat)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to list states")
	}
	states := StateList{}
	if err := json.Unmarshal(response, &states); err != nil {
		return 0, err
	}
	for _, s := range states.States {
		if strings.EqualFold(s.StateName, state) {
			log.Printf("State Details - ID: %d, Name: %s", s.StateID, s.StateName)
			return s.StateID, nil
		}
	}
	return 0, errors.New("Invalid state name passed")
}

func getDistrictIDByName(stateID int, district string) (int, error) {
	response, err := queryServer(fmt.Sprintf(listDistrictsURLFormat, stateID))
	if err != nil {
		return 0, errors.Wrap(err, "Failed to list states")
	}
	dl := DistrictList{}
	if err := json.Unmarshal(response, &dl); err != nil {
		return 0, err
	}
	for _, d := range dl.Districts {
		if strings.EqualFold(d.DistrictName, district) {
			log.Printf("District Details - ID: %d, Name: %s", d.DistrictID, d.DistrictName)
			return d.DistrictID, nil
		}
	}
	return 0, errors.New("Invalid district name passed")
}

func searchByStateDistrict(age int, state, district string) error {
	var err1 error
	if stateID == 0 {
		stateID, err1 = getStateIDByName(state)
		if err1 != nil {
			return err1
		}
	}
	if districtID == 0 {
		districtID, err1 = getDistrictIDByName(stateID, district)
		if err1 != nil {
			return err1
		}
	}
	response, err := queryServer(fmt.Sprintf(calendarByDistrictURLFormat, districtID, date))
	if err != nil {
		return errors.Wrap(err, "Failed to fetch appointment sessions")
	}
	return getAvailableSessions(response, age)
}

// isPreferredAvailable checks for availability of preferences
func isPreferredAvailable(current, preference string) bool {
	if preference == "" {
		return true
	} else {
		return strings.EqualFold(current, preference)
	}
}

// isDoseAvailable checks for availability of required dose and quantity
func isDoseAvailable(current Sessions, quantity, dose int) bool {
	if dose == 1 {
		return current.AvailableCapacityDose1 >= quantity
	} else {
		return current.AvailableCapacityDose2 >= quantity
	}
}

func getAvailableSessions(response []byte, age int) error {
	if response == nil {
		log.Printf("Received unexpected response, rechecking after %v seconds", interval)
		return nil
	}
	appointments := Appointments{}
	err := json.Unmarshal(response, &appointments)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 1, 8, 1, '\t', 0)
	for _, center := range appointments.Centers {
		if !isPreferredAvailable(center.FeeType, fee) {
			continue
		}
		for _, s := range center.Sessions {
			if s.MinAgeLimit <= age && isDoseAvailable(s, quantity, dose) && isPreferredAvailable(s.Vaccine, vaccine) {
				fmt.Fprintln(w, fmt.Sprintf("Center\t%s", center.Name))
				fmt.Fprintln(w, fmt.Sprintf("State\t%s", center.StateName))
				fmt.Fprintln(w, fmt.Sprintf("District\t%s", center.DistrictName))
				fmt.Fprintln(w, fmt.Sprintf("PinCode\t%d", center.Pincode))
				fmt.Fprintln(w, fmt.Sprintf("Fee\t%s", center.FeeType))
				if len(center.VaccineFees) != 0 {
					fmt.Fprintln(w, fmt.Sprintf("Vaccine\t"))
				}
				for _, v := range center.VaccineFees {
					fmt.Fprintln(w, fmt.Sprintf("\tName\t%s", v.Vaccine))
					fmt.Fprintln(w, fmt.Sprintf("\tFees\t%s", v.Fee))
				}
				fmt.Fprintln(w, fmt.Sprintf("Sessions\t"))
				fmt.Fprintln(w, fmt.Sprintf("\tDate\t%s", s.Date))
				fmt.Fprintln(w, fmt.Sprintf("\tAvailableCapacity\t%d", s.AvailableCapacity))
				fmt.Fprintln(w, fmt.Sprintf("\tAvailableCapacityDose1\t%d", s.AvailableCapacityDose1))
				fmt.Fprintln(w, fmt.Sprintf("\tAvailableCapacityDose2\t%d", s.AvailableCapacityDose2))
				fmt.Fprintln(w, fmt.Sprintf("\tMinAgeLimit\t%d", s.MinAgeLimit))
				fmt.Fprintln(w, fmt.Sprintf("\tVaccine\t%s", s.Vaccine))
				fmt.Fprintln(w, fmt.Sprintf("\tSlots"))
				for _, slot := range s.Slots {
					fmt.Fprintln(w, fmt.Sprintf("\t\t%s", slot))
				}
				fmt.Fprintln(w, "-----------------------------")
			}
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if buf.Len() == 0 {
		log.Printf("No slots available, rechecking after %v seconds", interval)
		return nil
	}
	log.Print("Found available slots, sending email")
	// log.Print(buf.String())
	return sendMail(email, password, buf.String())
}

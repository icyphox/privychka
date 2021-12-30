package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Habit struct {
	Time     time.Time `json:"time"`
	Activity string    `json:"activity"`
	Notes    string    `json:"notes,omitempty"`
}

func dateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func ReadTSV(fname string) ([]Habit, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(f)
	r.Comma = '\t'
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	habits := []Habit{}

	for _, record := range records {
		h := Habit{}
		h.Time, _ = time.Parse(time.RFC1123, record[0])
		h.Activity = record[1]
		if len(record) == 2 {
			h.Notes = ""
		} else {
			h.Notes = record[2]
		}
		habits = append(habits, h)
	}

	return habits, nil
}

func getTodaysHabits(habits []Habit) []Habit {
	todays := []Habit{}
	for _, h := range habits {
		if dateEqual(h.Time, time.Now()) {
			todays = append(todays, h)
		}
	}
	return todays
}

func (h Habit) String() string {
	t := h.Time.Format(time.RFC1123)
	return fmt.Sprintf(
		"time: %s  activity: %s  notes:  %s",
		t,
		h.Activity,
		h.Notes,
	)
}

func (h Habit) WriteTSV(fname string) error {
	records := []string{}

	f, err := os.OpenFile(fname, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	w.Comma = '\t'
	t := h.Time.Format(time.RFC1123)

	records = append(records, t, h.Activity, h.Notes)
	if err := w.Write(records); err != nil {
		return err
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	defer f.Close()

	return nil
}

func getKey(r *http.Request) (string, error) {
	var key string
	header := r.Header.Get("Authorization")
	if len(header) == 0 {
		return "", errors.New("missing Authorization header")
	}
	key = strings.Split(header, " ")[1]
	return key, nil
}

func main() {
	hFile := flag.String("f", "./habits.tsv", "csv file to store habit data")
	secretKey := flag.String("k", "", "auth key to be passed as bearer token")
	flag.Parse()

	if _, err := os.Stat(*hFile); errors.Is(err, os.ErrNotExist) {
		_, err := os.Create(*hFile)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		h := Habit{}
		key, err := getKey(r)
		if err != nil {
			log.Printf("error: %v\n", err)
			w.WriteHeader(401)
			return
		}

		if *secretKey != key {
			log.Printf("incorrect key: %v\n", key)
			w.WriteHeader(401)
			return
		}

		json.NewDecoder(r.Body).Decode(&h)
		log.Printf(h.String())

		if err := h.WriteTSV(*hFile); err != nil {
			log.Printf("error: %v\n", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(204)
	})

	http.HandleFunc("/today", func(w http.ResponseWriter, r *http.Request) {
		key, err := getKey(r)
		if err != nil {
			log.Printf("error: %v\n", err)
			w.WriteHeader(401)
			return
		}

		if *secretKey != key {
			log.Printf("incorrect key: %v\n", key)
			w.WriteHeader(401)
			return
		}

		habits, err := ReadTSV(*hFile)
		if err != nil {
			log.Printf("error: %v\n", err)
			w.WriteHeader(500)
			return
		}
		todays := getTodaysHabits(habits)
		json.NewEncoder(w).Encode(todays)
		return
	})

	http.ListenAndServe(":8585", nil)
}

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

func (h Habit) String() string {
	t := h.Time.Format(time.RFC1123Z)
	return fmt.Sprintf(
		"time: %s  activity: %s  notes:  %s",
		t,
		h.Activity,
		h.Notes,
	)
}

func (h Habit) WriteTSV(fname string) error {
	record := []string{}

	f, err := os.OpenFile(fname, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	w.Comma = '\t'
	t := h.Time.Format(time.RFC1123)

	record = append(record, t, h.Activity, h.Notes)
	if err := w.Write(record); err != nil {
		return err
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	defer f.Close()

	return nil
}

func getKey(r *http.Request) string {
	var key string
	header := r.Header.Get("Authorization")
	key = strings.Split(header, " ")[1]
	return key
}

func main() {
	hFile := *flag.String("f", "./habits.tsv", "csv file to store habit data")
	secretKey := *flag.String("key", "secret", "auth key to be passed as bearer token")
	flag.Parse()

	if _, err := os.Stat(hFile); errors.Is(err, os.ErrNotExist) {
		_, err := os.Create(hFile)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		h := Habit{}
		key := getKey(r)
		if secretKey != key {
			log.Printf("incorrect key: %v\n", key)
			w.WriteHeader(401)
			return
		}
		json.NewDecoder(r.Body).Decode(&h)
		log.Printf(h.String())

		if err := h.WriteTSV(hFile); err != nil {
			log.Printf("error: %v\n", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(204)
	})

	http.ListenAndServe(":8585", nil)
}

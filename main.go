package main

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

//go:embed web/*
var assets embed.FS

func app(m *Model) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/style.css" && r.URL.Path != "/app.js" {
			http.NotFound(w, r)
			return
		}
		name := "index.html"
		kind := "text/html; charset=utf-8"
		if r.URL.Path == "/style.css" {
			name = "style.css"
			kind = "text/css"
		}
		if r.URL.Path == "/app.js" {
			name = "app.js"
			kind = "text/javascript"
		}
		data, _ := assets.ReadFile("web/" + name)
		w.Header().Set("Content-Type", kind)
		w.Write(data)
	})
	mux.HandleFunc("/api/model", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"fingerprint": m.Fingerprint, "source": m.Source, "demo": m.Demo, "train": m.Train, "calibration": m.Calibration, "test": m.Test, "metrics": m.Metrics, "ranges": m.Ranges, "version": "stridewise-v1", "target": "Illustrative mobility index at week 12, 0 to 100"})
	})
	mux.HandleFunc("/api/predict", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Use POST", 405)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var input struct {
			Age      *float64 `json:"age"`
			Mobility *float64 `json:"mobility"`
			Pain     *float64 `json:"pain"`
			Weeks    *float64 `json:"weeks"`
			Coverage int      `json:"coverage"`
		}
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if err := d.Decode(&input); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		if d.Decode(&struct{}{}) != io.EOF {
			http.Error(w, "Expected one JSON object", 400)
			return
		}
		if input.Age == nil || input.Mobility == nil || input.Pain == nil || input.Weeks == nil {
			http.Error(w, "All baseline fields are required", 400)
			return
		}
		p := Profile{*input.Age, *input.Mobility, *input.Pain, *input.Weeks}
		if e := validate(p); e != nil {
			http.Error(w, e.Error(), 400)
			return
		}
		if input.Coverage != 80 && input.Coverage != 90 && input.Coverage != 95 {
			http.Error(w, "Coverage must be 80, 90, or 95", 400)
			return
		}
		if !m.inRange(p) {
			http.Error(w, "Profile is outside the observed training ranges. Review the cohort ranges in Model evidence.", 422)
			return
		}
		writeJSON(w, m.predict(p, input.Coverage))
	})
	mux.HandleFunc("/api/example.csv", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="stridewise-synthetic.csv"`)
		c := csv.NewWriter(w)
		c.Write([]string{"patient_id", "age", "baseline_mobility", "baseline_pain", "weeks_since_event", "mobility_week12"})
		for _, row := range synthetic() {
			values := []string{row.ID}
			for _, v := range []float64{row.Age, row.Mobility, row.Pain, row.Weeks, row.Outcome} {
				values = append(values, strconv.FormatFloat(v, 'f', 6, 64))
			}
			c.Write(values)
		}
		c.Flush()
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; object-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		mux.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "Listen address")
	dataset := flag.String("data", "", "Local retrospective CSV path")
	evaluate := flag.Bool("evaluate", false, "Print evaluation and exit")
	flag.Parse()
	rows := synthetic()
	source := "1,200 synthetic records · seed 42"
	demo := true
	if *dataset != "" {
		f, e := os.Open(*dataset)
		if e != nil {
			log.Fatal(e)
		}
		rows, e = readCSV(f)
		f.Close()
		if e != nil {
			log.Fatal(e)
		}
		source = "Local retrospective CSV · user supplied, unverified provenance"
		demo = false
	}
	m, e := fit(rows, source, demo)
	if e != nil {
		log.Fatal(e)
	}
	if *evaluate {
		b, _ := json.MarshalIndent(map[string]any{"fingerprint": m.Fingerprint, "source": m.Source, "demo": m.Demo, "training": m.Train, "calibration": m.Calibration, "test": m.Test, "metrics": m.Metrics}, "", "  ")
		fmt.Println(string(b))
		return
	}
	s := &http.Server{Addr: *addr, Handler: app(m), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("Stridewise at http://%s", *addr)
	log.Fatal(s.ListenAndServe())
}

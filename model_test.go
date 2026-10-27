package main

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func demoModel(t *testing.T) *Model {
	t.Helper()
	m, e := fit(synthetic(), "synthetic", true)
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func TestPredictionBehavior(t *testing.T) {
	m := demoModel(t)
	p := Profile{58, 42, 4, 1}
	a, b, c := m.predict(p, 80), m.predict(p, 90), m.predict(p, 95)
	if a.Lower > b.Lower && b.Lower >= c.Lower && a.Upper < b.Upper && b.Upper <= c.Upper {
	} else {
		t.Fatal("coverage must widen intervals")
	}
	if m.predict(Profile{58, 62, 4, 1}, 90).Estimate <= b.Estimate {
		t.Fatal("baseline mobility should affect estimate")
	}
	if m.Metrics.MAE >= m.Metrics.BaselineMAE {
		t.Fatal("model must beat mean baseline on demo test set")
	}
	if m.Metrics.Coverage < 80 || m.Metrics.Coverage > 100 {
		t.Fatal("unexpected held-out coverage")
	}
	if m.Train != 720 || m.Calibration != 240 || m.Test != 240 {
		t.Fatal("split counts")
	}
	n := demoModel(t)
	if n.Metrics != m.Metrics || n.predict(p, 90) != b {
		t.Fatal("not reproducible")
	}
}
func TestConformalRank(t *testing.T) {
	m := &Model{Residuals: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}}
	m.Beta[0] = 50
	p := m.predict(Profile{}, 80)
	if p.Lower != 42 || p.Upper != 58 {
		t.Fatal(p)
	}
	p = m.predict(Profile{}, 95)
	if p.Lower != 0 || p.Upper != 100 {
		t.Fatal("insufficient calibration must yield full outcome interval")
	}
}
func TestCSVValidation(t *testing.T) {
	for _, body := range []string{"patient_id,age\n", "patient_id,age,baseline_mobility,baseline_pain,weeks_since_event,mobility_week12\na,NaN,30,4,1,50\n", "patient_id,age,baseline_mobility,baseline_pain,weeks_since_event,mobility_week12\na,50,30,4,1,50\na,50,30,4,1,50\n"} {
		if _, e := readCSV(strings.NewReader(body)); e == nil {
			t.Fatal("accepted invalid CSV")
		}
	}
	if validate(Profile{math.NaN(), 1, 1, 1}) == nil {
		t.Fatal("NaN accepted")
	}
}
func TestHTTPBoundaries(t *testing.T) {
	h := app(demoModel(t))
	cases := []struct {
		method, path, body string
		status             int
	}{{"GET", "/", "", 200}, {"GET", "/api/model", "", 200}, {"GET", "/missing", "", 404}, {"GET", "/api/predict", "", 405}, {"POST", "/api/predict", `{"age":58,"mobility":42,"pain":4,"weeks":1,"coverage":90}`, 200}, {"POST", "/api/predict", `{"age":58,"mobility":42,"pain":4,"weeks":1,"coverage":99}`, 400}, {"POST", "/api/predict", `{"age":58,"mobility":42,"pain":4,"weeks":1,"coverage":90,"extra":1}`, 400}, {"POST", "/api/predict", `{"age":58,"mobility":42,"pain":4,"coverage":90}`, 400}, {"POST", "/api/predict", `{"age":58,"mobility":0,"pain":4,"weeks":1,"coverage":90}`, 422}, {"POST", "/api/predict", `{"age":58,"mobility":42,"pain":4,"weeks":1,"coverage":90} {}`, 400}, {"POST", "/api/predict", strings.Repeat("x", 5000), 400}}
	for _, c := range cases {
		t.Run(c.method+c.path+c.body[:min(len(c.body), 15)], func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(c.method, c.path, strings.NewReader(c.body)))
			if w.Code != c.status {
				t.Fatalf("got %d expected %d: %s", w.Code, c.status, w.Body)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cache header")
			}
		})
	}
}
func TestCSVDownloadRoundTrip(t *testing.T) {
	h := app(demoModel(t))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/example.csv", nil))
	rows, e := readCSV(bytes.NewReader(w.Body.Bytes()))
	if e != nil || len(rows) != 1200 {
		t.Fatal(len(rows), e)
	}
	m, e := fit(rows, "local", false)
	if e != nil {
		t.Fatal(e)
	}
	w = httptest.NewRecorder()
	app(m).ServeHTTP(w, httptest.NewRequest("GET", "/api/model", nil))
	var v struct{ Demo bool }
	if json.Unmarshal(w.Body.Bytes(), &v) != nil || v.Demo {
		t.Fatal("real-data mode mislabeled")
	}
}

func TestIdentifiersAndPrecision(t *testing.T) {
	header := "patient_id,age,baseline_mobility,baseline_pain,weeks_since_event,mobility_week12\n"
	for _, id := range []string{" ", " P1", "P1 "} {
		if _, err := readCSV(strings.NewReader(header + id + ",58,42,4,1,60\n")); err == nil {
			t.Fatal("accepted padded or blank identifier")
		}
	}
	var body strings.Builder
	body.WriteString(header)
	for i := 0; i < 150; i++ {
		body.WriteString("P" + strconv.Itoa(i) + ",58.05,42.05,4.05,1.05,60\n")
	}
	rows, err := readCSV(strings.NewReader(body.String()))
	if err != nil {
		t.Fatal(err)
	}
	m, err := fit(rows, "precision", false)
	if err != nil {
		t.Fatal(err)
	}
	if !m.inRange(Profile{58.05, 42.05, 4.05, 1.05}) {
		t.Fatal("valid exact profile refused")
	}
	if len(m.Fingerprint) != 64 || m.Metrics.Eligible+m.Metrics.Refused != m.Test {
		t.Fatal("audit metadata missing")
	}
}

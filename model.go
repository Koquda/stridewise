package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

type Profile struct {
	Age      float64 `json:"age"`
	Mobility float64 `json:"mobility"`
	Pain     float64 `json:"pain"`
	Weeks    float64 `json:"weeks"`
}
type Record struct {
	ID string
	Profile
	Outcome float64
}
type Metrics struct {
	Eligible         int     `json:"eligible"`
	Refused          int     `json:"refused"`
	EligibleCoverage float64 `json:"eligible_coverage"`

	N           int     `json:"n"`
	MAE         float64 `json:"mae"`
	BaselineMAE float64 `json:"baseline_mae"`
	Coverage    float64 `json:"coverage"`
	Width       float64 `json:"width"`
}
type Model struct {
	Fingerprint string

	Beta                     [5]float64
	Residuals                []float64
	Train, Calibration, Test int
	Metrics                  Metrics
	Source                   string
	Demo                     bool
	Ranges                   [4][2]float64
	Rows                     []Record
}
type Prediction struct {
	Estimate float64 `json:"estimate"`
	Lower    float64 `json:"lower"`
	Upper    float64 `json:"upper"`
	Coverage int     `json:"coverage"`
	Change   float64 `json:"change"`
}

func features(p Profile) [5]float64 {
	return [5]float64{1, (p.Age - 50) / 30, (p.Mobility - 50) / 30, (p.Pain - 5) / 5, p.Weeks / 6}
}
func clamp(x float64) float64 { return math.Max(0, math.Min(100, x)) }
func validate(p Profile) error {
	values := []float64{p.Age, p.Mobility, p.Pain, p.Weeks}
	bounds := [][2]float64{{18, 90}, {0, 100}, {0, 10}, {0, 6}}
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < bounds[i][0] || v > bounds[i][1] {
			return fmt.Errorf("%s must be between %g and %g", []string{"Age", "Mobility", "Pain", "Weeks since event"}[i], bounds[i][0], bounds[i][1])
		}
	}
	return nil
}
func synthetic() []Record {
	rng := rand.New(rand.NewSource(42))
	rows := make([]Record, 1200)
	for i := range rows {
		p := Profile{18 + rng.Float64()*72, 10 + rng.Float64()*75, rng.Float64() * 10, rng.Float64() * 6}
		y := clamp(30 + .76*p.Mobility - .13*(p.Age-50) - 1.65*p.Pain - .9*p.Weeks + rng.NormFloat64()*7)
		rows[i] = Record{fmt.Sprintf("S%04d", i+1), p, y}
	}
	return rows
}
func readCSV(r io.Reader) ([]Record, error) {
	c := csv.NewReader(r)
	header, err := c.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	expected := []string{"patient_id", "age", "baseline_mobility", "baseline_pain", "weeks_since_event", "mobility_week12"}
	if len(header) != len(expected) {
		return nil, fmt.Errorf("expected exactly six columns")
	}
	for i, h := range expected {
		if header[i] != h {
			return nil, fmt.Errorf("column %d must be %s", i+1, h)
		}
	}
	seen := map[string]bool{}
	var rows []Record
	for line := 2; ; line++ {
		fields, e := c.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, fmt.Errorf("row %d: %w", line, e)
		}
		if len(rows) >= 100000 {
			return nil, fmt.Errorf("maximum 100000 records")
		}
		if strings.TrimSpace(fields[0]) == "" || fields[0] != strings.TrimSpace(fields[0]) || seen[fields[0]] {
			return nil, fmt.Errorf("row %d: patient_id must be nonempty and unique", line)
		}
		seen[fields[0]] = true
		var v [5]float64
		for i := range v {
			n, e := strconv.ParseFloat(fields[i+1], 64)
			if e != nil || math.IsNaN(n) || math.IsInf(n, 0) {
				return nil, fmt.Errorf("row %d: invalid numeric value", line)
			}
			v[i] = n
		}
		p := Profile{v[0], v[1], v[2], v[3]}
		if e := validate(p); e != nil {
			return nil, fmt.Errorf("row %d: %w", line, e)
		}
		if v[4] < 0 || v[4] > 100 {
			return nil, fmt.Errorf("row %d: outcome must be 0 to 100", line)
		}
		rows = append(rows, Record{fields[0], p, v[4]})
	}
	if len(rows) < 150 {
		return nil, fmt.Errorf("at least 150 unique complete records are required")
	}
	return rows, nil
}
func fit(rows []Record, source string, demo bool) (*Model, error) {
	shuffled := append([]Record(nil), rows...)
	rng := rand.New(rand.NewSource(17))
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	a, b := len(rows)*6/10, len(rows)*8/10
	m := &Model{Train: a, Calibration: b - a, Test: len(rows) - b, Source: source, Demo: demo, Rows: rows}
	var matrix [5][6]float64
	for i := range m.Ranges {
		m.Ranges[i] = [2]float64{math.Inf(1), math.Inf(-1)}
	}
	encoded, _ := json.Marshal(rows)
	m.Fingerprint = fmt.Sprintf("%x", sha256.Sum256(encoded))
	mean := 0.
	for _, r := range shuffled[:a] {
		x := features(r.Profile)
		mean += r.Outcome
		for i := 0; i < 5; i++ {
			for j := 0; j < 5; j++ {
				matrix[i][j] += x[i] * x[j]
			}
			matrix[i][5] += x[i] * r.Outcome
		}
		for i, v := range []float64{r.Age, r.Mobility, r.Pain, r.Weeks} {
			m.Ranges[i][0] = math.Min(m.Ranges[i][0], v)
			m.Ranges[i][1] = math.Max(m.Ranges[i][1], v)
		}
	}
	mean /= float64(a)
	for i := 1; i < 5; i++ {
		matrix[i][i] += .1
	}
	for i := 0; i < 5; i++ {
		pivot := i
		for j := i + 1; j < 5; j++ {
			if math.Abs(matrix[j][i]) > math.Abs(matrix[pivot][i]) {
				pivot = j
			}
		}
		matrix[i], matrix[pivot] = matrix[pivot], matrix[i]
		if math.Abs(matrix[i][i]) < 1e-10 {
			return nil, fmt.Errorf("training matrix is singular")
		}
		d := matrix[i][i]
		for k := i; k < 6; k++ {
			matrix[i][k] /= d
		}
		for j := 0; j < 5; j++ {
			if j != i {
				v := matrix[j][i]
				for k := i; k < 6; k++ {
					matrix[j][k] -= v * matrix[i][k]
				}
			}
		}
	}
	for i := range m.Beta {
		m.Beta[i] = matrix[i][5]
	}
	for _, r := range shuffled[a:b] {
		m.Residuals = append(m.Residuals, math.Abs(r.Outcome-m.point(r.Profile)))
	}
	sort.Float64s(m.Residuals)
	metrics := Metrics{N: m.Test}
	for _, r := range shuffled[b:] {
		p := m.predict(r.Profile, 90)
		if m.inRange(r.Profile) {
			metrics.Eligible++
			if r.Outcome >= p.Lower && r.Outcome <= p.Upper {
				metrics.EligibleCoverage++
			}
		} else {
			metrics.Refused++
		}
		metrics.MAE += math.Abs(p.Estimate - r.Outcome)
		metrics.BaselineMAE += math.Abs(mean - r.Outcome)
		metrics.Width += p.Upper - p.Lower
		if r.Outcome >= p.Lower && r.Outcome <= p.Upper {
			metrics.Coverage++
		}
	}
	n := float64(m.Test)
	metrics.MAE /= n
	metrics.BaselineMAE /= n
	metrics.Width /= n
	metrics.Coverage = 100 * metrics.Coverage / n
	if metrics.Eligible > 0 {
		metrics.EligibleCoverage = 100 * metrics.EligibleCoverage / float64(metrics.Eligible)
	}
	m.Metrics = metrics
	return m, nil
}
func (m *Model) point(p Profile) float64 {
	x := features(p)
	y := 0.
	for i, v := range x {
		y += v * m.Beta[i]
	}
	return clamp(y)
}
func (m *Model) predict(p Profile, coverage int) Prediction {
	rank := int(math.Ceil(float64(len(m.Residuals)+1) * float64(coverage) / 100))
	q := math.Inf(1)
	if rank <= len(m.Residuals) {
		q = m.Residuals[rank-1]
	}
	y := m.point(p)
	return Prediction{y, clamp(y - q), clamp(y + q), coverage, y - p.Mobility}
}
func (m *Model) inRange(p Profile) bool {
	for i, v := range []float64{p.Age, p.Mobility, p.Pain, p.Weeks} {
		if v < m.Ranges[i][0] || v > m.Ranges[i][1] {
			return false
		}
	}
	return true
}

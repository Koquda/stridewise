# Stridewise

A local recovery-outcome research app, written in Go. Estimate one week-12 mobility score and display its prediction interval. Explore held-out model evidence without losing sight of the dataset's limitations.

**The default dataset is synthetic. This is a research prototype, not validated clinical software.** The illustrative mobility index is not a validated instrument. No intermediate recovery curve, milestone date, or treatment effect is inferred.

## Run

Requires Go 1.22 or later. No packages, database, frontend build, or network service is required.

```sh
go run . -addr 127.0.0.1:8087
```

Open http://127.0.0.1:8087. The app binds to loopback by default. Submitted baseline profiles stay in memory and are not saved or logged. The app makes no analytics or remote asset requests.

On macOS hosts where Go 1.22 reports `missing LC_UUID load command`, use the system external linker.

```sh
go run -ldflags=-linkmode=external . -addr 127.0.0.1:8087
```

## Use a retrospective CSV

Prepare a local CSV with this exact column order.

```csv
patient_id,age,baseline_mobility,baseline_pain,weeks_since_event,mobility_week12
P001,58,42,4,1,61
```

The file needs at least 150 unique, complete records. Ages must be 18–90, mobility values 0–100, pain 0–10, and baseline timing 0–6 weeks after the event. Each row represents one person. Use pseudonymous IDs. Keep local files in the ignored `data/` directory.

```sh
go run . -data data/cohort.csv -addr 127.0.0.1:8087
```

The parser rejects blank or padded IDs, duplicate IDs, non-finite values, missing values, unexpected columns, and more than 100,000 records. It does not verify source consent, assessment dates, instrument validity, cohort eligibility, or that predictors were actually collected before the outcome. The researcher must establish those facts. Use a single defined event and outcome instrument across the cohort. Do not treat the numeric example as a clinical scoring protocol.

The browser refuses predictions outside observed training ranges. This check does not detect unusual combinations inside those ranges. Production use also needs authentication, encrypted transport, governance, monitoring, and validation. Local files are not encrypted by this app.

## Reproduce the evaluation

```sh
go run . -evaluate
go test -race ./...
go vet ./...
go build ./...
```

Add `-ldflags=-linkmode=external` to `go run`, `go test`, and `go build` on the affected macOS toolchain. Run `go run . -data data/cohort.csv -evaluate` to evaluate a local cohort.

The demonstration uses 1,200 synthetic records, generation seed 42, and shuffle seed 17. A fixed 60/20/20 split produces 720 training, 240 calibration, and 240 test records. Scaling constants and ridge penalty 0.1 are fixed before evaluation. The intercept is unpenalized. The mean baseline uses only the training outcomes.

| Synthetic test metric | Result |
| --- | ---: |
| Mean absolute error | 5.21 points |
| Training-mean baseline error | 15.79 points |
| Observed coverage at 90% nominal | 93.33% |
| Mean interval width | 23.66 points |

These are observed results from one simulated test set, not evidence of real-patient accuracy. The generator deliberately has a simple relationship close to the fitted model. Real data will be harder.

## Understand the interval

Ridge regression predicts the endpoint. Split conformal calibration sorts absolute residuals on a separate set and selects rank `ceil((n+1)*coverage)`. If the rank exceeds the calibration count, the interval becomes the full valid outcome range. Predictions and interval endpoints are clipped to 0–100. The UI supports 80%, 90%, and 95% nominal coverage.

The method targets marginal population coverage under exchangeability. It does not promise individual or subgroup coverage. See [Lei et al., Distribution-Free Predictive Inference for Regression](https://www.stat.berkeley.edu/~ryantibs/papers/conformal-jasa.pdf).

Repeated IDs cannot cross partitions because the importer requires one row per unique ID. The researcher must ensure each person has exactly one ID. IDs are not predictors. The random split is a demonstration default. Real research needs temporal or site holdouts, missing-outcome analysis, subgroup evaluation, and external validation. Changing row order changes partition membership. Preserve the original input file when reproducing an analysis.

## Maintain the project

`model.go` owns the data contract, generation, parsing, fit, calibration, and evaluation. `main.go` owns HTTP validation and startup. `web/` contains embedded HTML, CSS, and vanilla JavaScript. Go serves the entire app in one binary. No JavaScript runtime is needed to build or run it.

Tests cover prediction response, interval ordering, conformal rank, reproducibility, CSV validation and round trips, provenance, and HTTP boundaries. GitHub Actions runs race tests, vet, and build on Linux.

The interface uses system typography, restrained green materials, keyboard focus, an accessible interval chart, light and dark themes, responsive layouts, and reduced-motion and contrast preferences. See [the build plan](docs/PLAN.md), [architecture decision](docs/ARCHITECTURE.md), and [verification record](docs/VERIFICATION.md).

The model API exposes a SHA-256 fingerprint of the parsed, ordered input records. Preserve the source CSV and the model version to reproduce a run. The evidence page reports how many held-out profiles would be refused by serving ranges and their eligible-subset coverage. Main metrics include all test records.

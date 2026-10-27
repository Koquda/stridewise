# Verification record

Verified locally on 27 October 2026 with Go 1.22.2 on macOS arm64. External linking was required by this installed Go toolchain.

## Automated checks

- `GOCACHE=/private/tmp/stridewise-go-cache go test -ldflags=-linkmode=external -race ./...` passed.
- `GOCACHE=/private/tmp/stridewise-go-cache go vet ./...` passed.
- `GOCACHE=/private/tmp/stridewise-go-cache go build -ldflags=-linkmode=external` passed.
- Tests verify prediction response, interval ordering, finite-sample rank, deterministic evaluation, CSV round trips, malformed requests, required fields, supported coverage, range refusals, provenance, exact decimal profiles, and ID validation.

## Interactive browser checks

The running Go server was inspected in the in-app browser.

- Demo profile produced 53.6 points with a 90% interval of 41.7 to 65.4.
- Selecting 95% widened the interval to 39.5 to 67.6.
- Changing age from 58 to 68 and submitting changed the estimate to 52.6.
- After the stale-state fix, editing the form removed the estimate and displayed the update prompt.
- The evidence page displayed model metrics, split counts, and serving eligibility. Of 240 test records, 237 were eligible and 3 would be refused. Eligible coverage was 93.7%.
- A local synthetic CSV fixture with 150 constant decimal profiles loaded in imported-cohort mode. No graph appeared before submission. Exact values such as age 58.05 submitted successfully and produced a result.
- Layouts were visually inspected at 390 by 844 and 1440 by 1000. No horizontal overflow was observed on the narrow layout.
- Light and dark appearance were visually inspected. No warnings or errors appeared in the browser console during this check.

Reduced-motion, transparency, and contrast rules exist in CSS. They were not independently emulated. Physical-device testing and a full accessibility audit were not performed.

## Independent review

An independent GPT-6 Astra reviewer checked the model, boundary handling, UI state, comments, and architectural alternatives. It found no numerical train/test leakage. Its actionable findings led to fixes for stale results, invented pre-prediction chart marks, decimal input precision, and padded IDs. Additional audit findings led to dataset fingerprints and eligible-subset coverage reporting.

## Open validity work

No real patient dataset was supplied. The delivered default is a synthetic demonstration. Real-world accuracy, outcome validity, external calibration, fairness, and suitability for clinical decisions remain unverified. CSV validation cannot prove consent, source measurement timing, or that one ID corresponds to one person.

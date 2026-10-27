# Build plan

- [x] Read the Principles section of poteto-mode.
- [x] Phase A: Frame. Deliver a runnable Go research app with one 12-week outcome, honest data provenance, prediction intervals, and held-out evaluation.
- [x] Phase B: Design the workflow. Compare server-rendered and API-led alternatives before implementing.
- [x] Phase C: Run the loop. Build the model and verify numerical behavior. Build the server and verify boundaries. Build the interface and inspect it in a browser.
- [x] Phase D: Keep the audit trail. Record decisions and evidence in decisions.tsv.
- [x] Phase E: Verify and hand back. Run tests, race detector, vet, build, browser checks, and publish source.

## Throughput checkpoint

Three implementation units. Model and CSV contract, HTTP and interface, verification and delivery. Primary unknown is dataset availability. No patient dataset was supplied. Synthetic demonstration is the default and is labeled throughout. Real retrospective CSV can be loaded locally at startup. Clinical validity remains outside this prototype.

## Architecture contract

Record contains ID, age, baseline mobility, baseline pain, weeks since event, and week-12 mobility. All predictors are available at the baseline observation. One record per person. Model contains fitted coefficients, calibration residuals, training ranges, provenance, and held-out evaluation. HTTP handlers read an immutable model. No patient input is persisted or logged. Only the week-12 outcome is modeled. The interface must not invent intermediate recovery trajectories or milestone dates.

## Verification predicate

Fresh checkout runs with Go alone. Predictions respond to inputs. Intervals widen with requested coverage. Invalid inputs and out-of-range requests are rejected. Reported metrics come from untouched test records. CSV validation rejects duplicates, missing fields, and non-finite values. Desktop and narrow layouts work. Repository contains reproducible tests and CI.

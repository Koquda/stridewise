# Architecture decision

Two independent design candidates compared server-rendered forms with a stateless JSON API and an embedded frontend. The API-led design is the base. It gives coverage controls immediate feedback while keeping all inference in Go. The server-rendered candidate contributed the explicit refusal to draw untrained recovery trajectories.

One immutable model owns coefficients, residuals, ranges, provenance, and held-out metrics. HTTP handlers parse external input and call the model. The browser never calculates prediction intervals. Submitted profiles are not persisted. Baseline and outcome are distinct fields in the record, and labels never enter the predictor feature vector.

The package stays small rather than adding an application framework or a second build system. Splitting numerical and HTTP ownership into separate files gives the next maintainer a short path through each behavior. An independent reviewer evaluated auditability, maintenance cost, interaction quality, and provenance.

The retrospective CSV option is deliberately local. No upload service, account system, hosted deployment, or claim of clinical readiness is included. The schema is illustrative until a researcher maps a documented clinical instrument and cohort protocol.

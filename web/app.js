const $ = (id) => document.getElementById(id);
let model,
  lastProfile,
  requestNumber = 0;
function page(name) {
  document
    .querySelectorAll(".page")
    .forEach((el) => (el.hidden = el.id !== name));
  document.querySelectorAll(".nav").forEach((el) => {
    el.classList.toggle("active", el.dataset.page === name);
    if (el.dataset.page === name) el.setAttribute("aria-current", "page");
    else el.removeAttribute("aria-current");
  });
  $("crumb").textContent = {
    estimate: "Recovery estimate",
    evidence: "Model evidence",
    data: "Data & methodology",
  }[name];
  history.replaceState(null, "", `#${name}`);
}
document
  .querySelectorAll("[data-page]")
  .forEach((el) => el.addEventListener("click", () => page(el.dataset.page)));
document.querySelectorAll("[data-goto]").forEach((el) =>
  el.addEventListener("click", () => {
    page(el.dataset.goto);
    $("main").scrollIntoView();
  }),
);
$("theme").addEventListener("click", () =>
  document.body.classList.toggle("dark"),
);
if (matchMedia("(prefers-color-scheme: dark)").matches)
  document.body.classList.add("dark");
for (const id of ["mobility", "pain"])
  $(id).addEventListener("input", () => {
    $(id + "-value").textContent =
      `${$(id).value} / ${id === "pain" ? 10 : 100}`;
  });
$("profile").addEventListener("input", () => {
  requestNumber++;
  lastProfile = undefined;
  $("result").hidden = true;
  $("result-empty").hidden = false;
  $("result-empty").textContent =
    "Profile changed. Estimate again to update the result.";
  document.querySelector(".primary").removeAttribute("aria-busy");
});
$("profile").addEventListener("submit", async (e) => {
  e.preventDefault();
  if (!$("profile").reportValidity()) return;
  await estimate({
    age: +$("age").value,
    mobility: +$("mobility").value,
    pain: +$("pain").value,
    weeks: +$("weeks").value,
  });
});
document.querySelectorAll("[name=coverage]").forEach((el) =>
  el.addEventListener("change", () => {
    requestNumber++;
    if (lastProfile) estimate(lastProfile);
  }),
);
function y(v) {
  return 145 - v * 1.2;
}
async function estimate(profile) {
  const number = ++requestNumber;
  $("result").hidden = true;
  $("result-empty").hidden = false;
  $("result-empty").textContent = "Calculating the estimate…";
  const coverage = +document.querySelector("[name=coverage]:checked").value;
  const button = document.querySelector(".primary");
  button.setAttribute("aria-busy", "true");
  try {
    const response = await fetch("/api/predict", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...profile, coverage }),
    });
    if (!response.ok) throw new Error(await response.text());
    const p = await response.json();
    if (number !== requestNumber) return;
    lastProfile = { ...profile };
    $("error").textContent = "";
    $("result").hidden = false;
    $("result-empty").hidden = true;
    $("score").textContent = p.estimate.toFixed(1);
    $("change").textContent = (p.change >= 0 ? "+" : "") + p.change.toFixed(1);
    $("interval").textContent = `${p.lower.toFixed(1)} – ${p.upper.toFixed(1)}`;
    $("interval-label").textContent = `${p.coverage}% prediction interval`;
    $("interval-explanation").textContent =
      `Targets ${p.coverage}% coverage across exchangeable cases. This is a population-level property, not an individual guarantee.${model.demo ? " Results describe simulated data only." : ""}`;
    $("baseline-dot").setAttribute("cy", y(profile.mobility));
    $("estimate-dot").setAttribute("cy", y(p.estimate));
    $("interval-band").setAttribute("y", y(p.upper));
    $("interval-band").setAttribute(
      "height",
      Math.max(1, (p.upper - p.lower) * 1.2),
    );
    $("interval-line").setAttribute("y1", y(p.upper));
    $("interval-line").setAttribute("y2", y(p.lower));
    $("chart-desc").textContent =
      `Baseline mobility ${profile.mobility}. Week-12 estimate ${p.estimate.toFixed(1)}, with ${coverage}% prediction interval ${p.lower.toFixed(1)} to ${p.upper.toFixed(1)}. No intermediate trajectory is modeled.`;
  } catch (e) {
    if (number === requestNumber) {
      $("error").textContent = e.message;
      $("result").hidden = true;
      $("result-empty").hidden = false;
    }
  } finally {
    if (number === requestNumber) button.removeAttribute("aria-busy");
  }
}
async function init() {
  try {
    const response = await fetch("/api/model");
    if (!response.ok) throw new Error("Model information could not be loaded.");
    model = await response.json();
    document
      .querySelectorAll(".provenance")
      .forEach(
        (el) =>
          (el.textContent = model.demo
            ? "Synthetic demo"
            : "Retrospective research"),
      );
    $("data-source").textContent =
      model.source + " · Dataset SHA-256 " + model.fingerprint;
    $("eligibility").textContent =
      `Of ${model.test} test records, ${model.metrics.eligible} fall within serving ranges and ${model.metrics.refused} would be refused. Observed 90% interval coverage among eligible test records is ${model.metrics.eligible ? model.metrics.eligible_coverage.toFixed(1) + "%" : "not available because no test records are eligible"}. Main metrics include all test records.`;
    if (!model.demo) {
      $("notice-text").textContent =
        "Local retrospective research. Dataset provenance and clinical validity have not been independently verified. Do not use to guide care.";
      $("data-title").textContent =
        "Your local cohort. Review its assumptions.";
    }
    $("cohort-summary").textContent = model.demo
      ? "1,200 simulated records. Zero real patients."
      : `${model.train + model.calibration + model.test} local cohort records. Unverified provenance.`;
    const metrics = [
      [model.metrics.mae.toFixed(2), "Mean absolute error, points"],
      [model.metrics.baseline_mae.toFixed(2), "Mean baseline error, points"],
      [model.metrics.coverage.toFixed(1) + "%", "Observed test coverage"],
      [model.metrics.width.toFixed(1), "Average interval width, points"],
    ];
    for (const [value, label] of metrics) {
      const div = document.createElement("div");
      div.className = "metric";
      const strong = document.createElement("strong");
      strong.textContent = value;
      const span = document.createElement("span");
      span.textContent = label;
      div.append(strong, span);
      $("metrics").append(div);
    }
    for (const key of ["train", "calibration", "test"]) {
      const span = document.createElement("span");
      span.textContent = `${key === "train" ? "Training" : key === "test" ? "Test" : "Calibration"} · ${model[key]} records`;
      $("splits").append(span);
    }
    const names = [
      "Age, years",
      "Baseline mobility, points",
      "Baseline pain, points",
      "Weeks since event",
    ];
    const ids = ["age", "mobility", "pain", "weeks"];
    model.ranges.forEach((r, i) => {
      const p = document.createElement("div");
      p.textContent = `${names[i]} · ${r[0].toFixed(2)} to ${r[1].toFixed(2)}`;
      $("ranges").append(p);
      const low = r[0],
        high = r[1];
      $(ids[i]).step = "any";
      $(ids[i]).min = low;
      $(ids[i]).max = high;
      $(ids[i]).value = Math.max(low, Math.min(high, +$(ids[i]).value));
      if (i === 1 || i === 2)
        $(ids[i] + "-value").textContent =
          `${$(ids[i]).value} / ${i === 1 ? 100 : 10}`;
    });
    page(
      ["estimate", "evidence", "data"].includes(location.hash.slice(1))
        ? location.hash.slice(1)
        : "estimate",
    );
    if (model.demo)
      await estimate({
        age: +$("age").value,
        mobility: +$("mobility").value,
        pain: +$("pain").value,
        weeks: +$("weeks").value,
      });
  } catch (e) {
    $("error").textContent = e.message;
  }
}
init();

const $ = (id) => document.getElementById(id);
const numberFormat = new Intl.NumberFormat("en", { maximumFractionDigits: 1 });
const formatNumber = (value) => numberFormat.format(value === 0 ? 0 : value);
function showSliderValue(id) {
  const input = $(id);
  const limit = id === "pain" ? 10 : 100;
  const value = formatNumber(Number(input.value));
  $(id + "-value").textContent = `${value} / ${limit}`;
  input.setAttribute("aria-valuetext", `${value} out of ${limit}`);
}
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
    showSliderValue(id);
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
    $("score").textContent = formatNumber(p.estimate);
    const displayedChange =
      (Math.round(p.estimate * 10) - Math.round(profile.mobility * 10)) / 10;
    $("change").textContent =
      (displayedChange > 0 ? "+" : "") + formatNumber(displayedChange);
    const changeIndicator = document.querySelector(".change");
    changeIndicator.querySelector("span").textContent =
      displayedChange > 0 ? "↗" : displayedChange < 0 ? "↘" : "→";
    changeIndicator.classList.toggle("decline", displayedChange < 0);
    $("interval").textContent =
      `${formatNumber(Math.floor(p.lower * 10) / 10)} – ${formatNumber(Math.ceil(p.upper * 10) / 10)}`;
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
      `Baseline mobility ${formatNumber(profile.mobility)}. Week-12 estimate ${formatNumber(p.estimate)}, with ${coverage}% prediction interval ${formatNumber(Math.floor(p.lower * 10) / 10)} to ${formatNumber(Math.ceil(p.upper * 10) / 10)}. No intermediate trajectory is modeled.`;
  } catch (e) {
    if (number === requestNumber) {
      $("error").textContent = e.message;
      $("result").hidden = true;
      $("result-empty").hidden = false;
      $("result-empty").textContent =
        "The estimate is unavailable. Review the error and try again.";
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
      [formatNumber(model.metrics.mae), "Mean absolute error, points"],
      [formatNumber(model.metrics.baseline_mae), "Mean baseline error, points"],
      [model.metrics.coverage.toFixed(1) + "%", "Observed test coverage"],
      [formatNumber(model.metrics.width), "Average interval width, points"],
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
      const input = $(ids[i]);
      let low = r[0],
        high = r[1];
      input.step = "any";
      if (i === 1 || i === 2) {
        const firstTenth = Math.ceil(low * 10) / 10;
        const lastTenth = Math.floor(high * 10) / 10;
        if (firstTenth <= lastTenth) {
          low = firstTenth;
          high = lastTenth;
          input.step = "0.1";
        } else {
          input.type = "number";
        }
      }
      input.min = low;
      input.max = high;
      const initial =
        input.type === "range"
          ? Math.round(Number(input.value) * 10) / 10
          : Number(input.value);
      input.value = Math.max(low, Math.min(high, initial));
      if (i === 1 || i === 2) showSliderValue(ids[i]);
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

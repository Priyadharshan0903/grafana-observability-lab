# 01 · SLO basics

Everything later (OpenSLO, Sloth, Pharos dashboards) is just machinery for
the ideas on this page. Get these solid first.

Dashboard: <http://localhost:3300/d/slo-basics> · Calculator: `cd budget && go run .`

---

## 1. SLI → SLO → error budget

| Term | What it is | Checkout example |
|---|---|---|
| **SLI** (indicator) | A ratio: `good events / valid events`, always between 0 and 1 | non-5xx requests / all requests |
| **SLO** (objective) | A target for the SLI over a window | 99.8% over 28 days |
| **Error budget** | `1 − SLO`: how much badness you're *allowed* | 0.2% of requests may fail |

Why a ratio and not "p99 latency < 300ms"? Ratios **add up** across time,
instances and regions: you can sum the good events and the total events and
divide. Percentiles can't be averaged or summed. So latency SLIs are also written as
ratios: *requests faster than 300ms ÷ all requests* (that's why the app's
histogram has an exact `le="0.3"` bucket).

```promql
# Availability SLI
sum(rate(http_requests_total{code!~"5.."}[1h])) / sum(rate(http_requests_total[1h]))

# Latency SLI
sum(rate(http_request_duration_seconds_bucket{le="0.3"}[1h]))
  / sum(rate(http_request_duration_seconds_count[1h]))
```

## 2. What 99.8% over 28 days actually allows

From `go run . -slo 0.998 -window 28d -rps 20`:

- 48.4M requests, of which **~96,800 may fail**
- the same budget as a **full outage of 1h20m**
- 99.9% over 30d → 43 minutes. Each extra "9" cuts the budget by 10×.

The budget is something you **spend**: on deploys, experiments and incidents.
If you never use any of it, the SLO is set too loose (or you're shipping too slowly).

## 3. Request-based vs timeslice SLOs

Same 99.8% number, two different ways to measure it:

| | Request-based (occurrences) | Timeslice |
|---|---|---|
| Unit counted | each request | each 15-minute slice |
| "Good" means | the request succeeded | ≥ 99% of requests **in that slice** succeeded |
| 28d at 99.8% allows | ~96.8k failed requests | **5 bad slices** (2688 × 0.2%) |
| Quiet hours | barely count (few requests) | count the same as peak hours |
| A 2-minute total outage | ~0.007% of budget | 1 whole bad slice ≈ **19%** of budget |

The Pharos dashboards use **15-minute timeslices**. So the budget isn't
"how many requests failed" but "how many 15-minute blocks were bad". A short sharp
outage costs much more under timeslices.

The dashboard's bottom-left panel draws the timeslices: one bar per 15 minutes,
1 = good and 0 = bad:

```promql
sum(rate(http_requests_total{code!~"5.."}[15m])) / sum(rate(http_requests_total[15m])) >= bool 0.99
```

`>= bool` turns the comparison into 1/0 instead of filtering. `avg_over_time(...[28d:15m])`
of that series is then "fraction of good slices": the timeslice SLI.

## 4. Burn rate

**Burn rate = observed error ratio ÷ allowed error ratio.**

```
burn rate = (1 − SLI over some window) / (1 − SLO)
```

- **1×** → you'll use exactly 100% of the budget by the end of the window. On pace.
- **2×** → budget gone in half the window (14 days of a 28d window).
- **14.4×** → a 30d budget gone in ~2 days.

Burn rate is what you **alert on**, not the SLI. "SLI dropped below 99.8%" fires
too late (budget already spent) and flaps. "We're burning 14× faster than
sustainable" fires early and means something.

## 5. Multi-window, multi-burn-rate alerts

These come from the Google SRE Workbook, and Sloth generates them for you (topic 3):

| Severity | Burn rate | Long window | Short window | Budget spent when it fires (30d) |
|---|---|---|---|---|
| Page | 14.4× | 1h | 5m | 2% |
| Page | 6× | 6h | 30m | 5% |
| Ticket | 1× | 3d | 6h | 10% |

**Why two windows?** The long window proves the problem is significant. The short
window proves it is *still happening*, so the alert clears soon after you fix it
instead of waiting an hour for the 1h average to recover. The alert fires only
when **both** windows are above the threshold.

Over a 28d window those percentages become 2.1% / 5.4% / 10.7% (run the calculator).

---

## Exercises (with the stack running)

Chaos control: `curl 'localhost:8091/admin/chaos?error_rate=X&slow_rate=Y'`

1. **Baseline.** Leave it ~10 min at defaults (0.1% errors). On the dashboard, with
   SLO = 0.998, what's the 1h burn rate? *(Expect ~0.5: you're failing half as much as allowed.)*
2. **Page-worthy incident.** `?error_rate=0.03`. Predict the burn rate first:
   `0.03 / 0.002 = 15×`. Watch the 5m line jump right away while the 1h line climbs slowly.
   How long until the 1h line crosses 14.4?
3. **Recovery.** Reset to `?error_rate=0.001`. The 5m line drops within minutes; the 1h line
   stays high. That gap is exactly why multi-window alerts need the short window.
4. **Change the SLO, not the system.** Same traffic, switch the SLO variable between
   0.99 and 0.999. Burn rate changes 10×. The SLO choice decides what counts as an incident.
5. **Timeslices hurt.** `?error_rate=0.5` for **2 minutes**, then reset. Wait for the
   15-minute slice to close. Request-based budget barely moves; the slice is marked bad
   (1 of the 5 bad slices allowed in 28 days).
6. **Latency SLI.** `?slow_rate=0.05` with errors at default. The availability SLI is perfect,
   the latency SLI drops to ~95%. Two SLIs, one service.
7. **Calculator.** `go run . -slo 0.995 -window 7d -rps 5`. How many bad 15-minute slices
   does a weekly 99.5% SLO allow? *(Answer: 3.36 → in practice 3.)*

## Check yourself

- Why are SLIs ratios and not percentiles?
- With a 99.8% / 28d timeslice SLO, how many bad slices can you afford?
- Burn rate is 3 over the last hour. Is that a page? What else do you need to know?
- Why does a multi-window alert need the short window?

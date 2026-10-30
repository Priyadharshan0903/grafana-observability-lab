# grafana-learnings

Hands-on path to building SLO dashboards like the Pharos/Grafana ones. A small
local stack mirrors the real pipeline: **SLO YAML → Sloth → Prometheus rules → Grafana**.

## Stack

```bash
cd stack && docker compose up -d --build
```

| Service | URL | What |
|---|---|---|
| Grafana | <http://localhost:3300> | no login (anonymous admin) |
| Prometheus | <http://localhost:9090> | scrapes `checkout` every 15s |
| checkout | <http://localhost:8091/metrics> | fake service with self-generated traffic |

Break things on purpose:

```bash
curl 'localhost:8091/admin/chaos?error_rate=0.05&slow_rate=0.1'   # inject errors / slowness
curl 'localhost:8091/admin/chaos?error_rate=0.001&slow_rate=0.005' # back to normal
```

Dashboards in `stack/grafana/dashboards/<topic>/*.json` load automatically
(the folder name becomes the Grafana folder). Edit in the UI, then **export the JSON back
into the repo**; UI-only edits are lost when the container is recreated.

Stop with `docker compose down` (add `-v` to wipe Prometheus/Grafana data).

## Path

1. [x] [SLO basics](01-slo-basics/): SLI/SLO/error budget, timeslices, burn rate
2. [ ] OpenSLO YAML
3. [ ] Sloth and recording rules
4. [ ] PromQL and Mimir
5. [ ] Dashboard JSON → Jinja → grafonnet / Go SDK
6. [ ] Pharos POP pipeline and `observability-artifacts`
7. [ ] Incident and alerting sources (P0/P1)
8. [ ] The 14 services

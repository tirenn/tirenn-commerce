# 🏗️ Tech Stack & Architecture: Tirenn Infrastructure & QA

This document outlines the shared network architecture, centralized logging pipeline, and automated testing architecture of `tirenn-infra`.

---

## 🌐 Network Topology & Inter-Service Communication

All services communicate over the high-speed Docker bridge network **`tirenn-net`**:

```
                       [ tirenn-net (Bridge Network) ]
                                      │
       ┌──────────────────────────────┼──────────────────────────────┐
       │                              │                              │
┌───────────────┐              ┌───────────────┐              ┌───────────────┐
│ tirenn-backend│              │tirenn-ai-serv.│              │tirenn-frontend│
│    (:8080)    │              │    (:8000)    │              │    (:3000)    │
└───────┬───────┘              └───────┬───────┘              └───────┬───────┘
        │                              │                              │
        └──────────────┬───────────────┘                              │
                       ▼                                              │
       ┌───────────────────────────────┐                              │
       │ tirenn-postgres (:5432)       │                              │
       │ tirenn-redis    (:6379)       │                              │
       │ tirenn-ollama   (:11434)      │                              │
       │ tirenn-loki     (:3100)       │◄─────────────────────────────┘
       │ tirenn-promtail               │   (Logs collected via Docker Socket)
       │ tirenn-grafana  (:3001)       │
       └───────────────────────────────┘
```

---

## 📊 Observability Architecture

1. **Docker Container STDOUT/STDERR** ➔ Scraped by **Promtail** via `/var/run/docker.sock`.
2. **Promtail** pushes structured logs with container labels to **Loki** (`http://loki:3100/loki/api/v1/push`).
3. **Grafana** automatically connects to Loki using `/etc/grafana/provisioning/datasources/datasources.yml`.

---

## 🧪 Automated Testing Architecture

- **Playwright Test Runner (`@playwright/test`)**:
  - Headless Chromium instances running in parallel across CPU workers.
  - Multi-page context isolation for both Shopper and Admin authentication states.

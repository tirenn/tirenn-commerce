# 🏗️ Tech Stack & Architecture: Tirenn Frontend

This document outlines the technical design, containerization, and styling architecture of `tirenn-frontend`.

---

## 💻 Technical Stack

| Layer | Technology | Details |
| :--- | :--- | :--- |
| **Language** | TypeScript 5.8 | Strict type checking, zero `any` policy where practical |
| **UI Library** | React 19 | Modern functional components, hooks, concurrent rendering |
| **Build Tool** | Vite 6 | Lightning-fast HMR and optimized Rollup production builds |
| **Styling** | Tailwind CSS 4 | Utility-first responsive design, modern design tokens |
| **Web Server** | Nginx Alpine | Multi-stage Docker build, lightweight static file server |

---

## 🐳 Container Architecture (`docker-compose.yml`)

```yaml
version: '3.8'

services:
  frontend:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: tirenn-frontend
    restart: always
    ports:
      - "3000:80"
    environment:
      - NODE_ENV=production
    networks:
      - tirenn-net

networks:
  tirenn-net:
    external: true
```

### Multi-Stage Dockerfile Lifecycle
1. **Builder Stage (`node:20-alpine`)**:
   - Copies `package*.json`, runs `npm install`.
   - Copies source and executes `npm run build` (producing `/app/dist`).
2. **Runtime Stage (`nginx:alpine`)**:
   - Copies `/app/dist` to `/usr/share/nginx/html`.
   - Injects `nginx.conf` with Single Page Application (SPA) routing (`try_files $uri /index.html`).

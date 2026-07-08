# syntax=docker/dockerfile:1

# ---- build ----
FROM node:24.18.0-alpine AS build
WORKDIR /src

# Separate dependency install so it is cached until the manifests change. The
# cache mount persists npm's download cache across builds.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY . .

# Production-only analytics: when set, the Yandex.Metrika snippet is baked
# into index.html at build time (see vite.config.ts); unset builds carry none.
ARG OBSERVER_METRIKA_ID

# Type-checks and emits the static bundle into dist/.
RUN npm run build

# ---- runtime ----
# Runs non-root and listens on 8080 out of the box, mirroring the backend
# images' non-root convention.
FROM nginxinc/nginx-unprivileged:1.30.3-alpine

COPY --from=build /src/dist /usr/share/nginx/html

# The official entrypoint envsubsts *.template files into /etc/nginx/conf.d at
# startup — that is what resolves ${OBSERVER_BACKEND_URL} in the proxy config.
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 8080

# Interval, timeout, and retries come from the compose healthcheck block; this
# supplies only the test. Probing / also verifies the SPA files are served.
HEALTHCHECK CMD wget -q --spider http://localhost:8080/ || exit 1

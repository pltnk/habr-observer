# syntax=docker/dockerfile:1
# check=skip=JSONArgsRecommended : the CMD needs shell form to expand the port.

FROM python:3.9-slim
WORKDIR /opt/frontend

# Resolve `from app import ...` from any working directory; flush logs straight
# to docker.
ENV PYTHONPATH=/opt/frontend
ENV PYTHONUNBUFFERED=true

# curl exists only for the HEALTHCHECK below. The non-root user mirrors the
# backend images; --create-home gives Streamlit a writable HOME.
RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home nonroot

# Separate dependency install so it is cached until requirements.txt changes.
# The cache mount persists pip's download cache across builds.
COPY requirements.txt ./
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r requirements.txt

# Overwrite Streamlit's bundled static assets with the app's branding
# (favicon, cover, localized index.html).
COPY app/static/ /usr/local/lib/python3.9/site-packages/streamlit/static/

COPY . .

# Build-time port, persisted as env for the HEALTHCHECK and CMD; compose passes
# OBSERVER_SL_PORT here and publishes the same value.
ARG SL_PORT=8501
EXPOSE ${SL_PORT}
ENV OBSERVER_SL_PORT=${SL_PORT}

USER nonroot

# Interval, timeout, and retries come from the compose healthcheck block; this
# supplies only the test.
HEALTHCHECK CMD curl --fail http://localhost:${OBSERVER_SL_PORT}/_stcore/health

# Shell form so the port variable expands.
CMD streamlit run run_app.py --server.address=0.0.0.0 --server.port=${OBSERVER_SL_PORT} --browser.gatherUsageStats=false

# syntax=docker/dockerfile:1

# 8.0.21+ refuse to start on Linux kernels 6.19+ (SERVER-121912) even where the
# incompatibility is fixed, so this pins the newest patch without that guard.
# Keep in sync with testMongoImage in the Go repository tests; switch to a
# floating mongo:8.0 once SERVER-125742 relaxes the guard.
FROM mongo:8.0.20

# Runs once, on the first start against an empty data dir, creating the app's
# database and collections from the container's environment.
COPY setup.sh /docker-entrypoint-initdb.d/

# ping needs no auth or database name. Interval, timeout, and retries come from
# the compose healthcheck block; this supplies only the test.
HEALTHCHECK CMD mongosh --quiet --eval 'db.runCommand("ping").ok' | grep -q 1

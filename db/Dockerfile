FROM mongo:6.0.7
COPY setup.sh /docker-entrypoint-initdb.d/
ARG OBSERVER_MONGO_DB=observer
HEALTHCHECK CMD mongosh ${OBSERVER_MONGO_DB} --eval 'db.runCommand("ping").ok' --quiet | grep 1

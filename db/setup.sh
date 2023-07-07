#!/bin/sh

# create db and collections for feeds and articles
DB="${OBSERVER_MONGO_DB:-observer}"
CMD="db.createCollection( '${OBSERVER_MONGO_ARTICLES:-articles}' ); db.createCollection( '${OBSERVER_MONGO_FEEDS:-feeds}' );"
mongosh "${DB}" --eval "${CMD}" -u "${OBSERVER_MONGO_USER:-default}" -p "${OBSERVER_MONGO_PASS:-default}" --authenticationDatabase admin

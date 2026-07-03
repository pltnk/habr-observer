import os

OBSERVER_BACKEND_URL = os.environ.get("OBSERVER_BACKEND_URL", "http://server:8080")
OBSERVER_FEED_CACHE_TTL = int(os.environ.get("OBSERVER_FEED_CACHE_TTL", 60))

import os

OBSERVER_FEEDS = {
    "Сутки": "https://habr.com/ru/rss/best/daily/?fl=ru",
    "Неделя": "https://habr.com/ru/rss/best/weekly/?fl=ru",
    "Месяц": "https://habr.com/ru/rss/best/monthly/?fl=ru",
    "Год": "https://habr.com/ru/rss/best/yearly/?fl=ru",
    "Всё время": "https://habr.com/ru/rss/best/alltime/?fl=ru",
}

OBSERVER_MONGO_USER = os.environ.get("OBSERVER_MONGO_USER", "default")
OBSERVER_MONGO_PASS = os.environ.get("OBSERVER_MONGO_PASS", "default")
OBSERVER_MONGO_DB = os.environ.get("OBSERVER_MONGO_DB", "observer")
OBSERVER_MONGO_ARTICLES = os.environ.get("OBSERVER_MONGO_ARTICLES", "articles")
OBSERVER_MONGO_FEEDS = os.environ.get("OBSERVER_MONGO_FEEDS", "feeds")
OBSERVER_MONGO_URI = f"mongodb://{OBSERVER_MONGO_USER}:{OBSERVER_MONGO_PASS}@db"
OBSERVER_AUTH_TOKEN = os.environ.get("OBSERVER_AUTH_TOKEN", "default")
OBSERVER_FEED_UPDATE_TIMEOUT = int(os.environ.get("OBSERVER_FEED_UPDATE_TIMEOUT", 600))
OBSERVER_FEED_CACHE_TTL = int(os.environ.get("OBSERVER_FEED_CACHE_TTL", 60))

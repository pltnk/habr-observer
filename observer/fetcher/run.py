import asyncio

from config import (
    OBSERVER_FEEDS,
    OBSERVER_MONGO_URI,
    OBSERVER_MONGO_DB,
    OBSERVER_MONGO_ARTICLES,
    OBSERVER_MONGO_FEEDS,
    OBSERVER_AUTH_TOKEN,
    OBSERVER_FEED_UPDATE_TIMEOUT,
)
from repository import MongoAsyncRepository
from .updater import FeedUpdater


async def update_feeds(repository: MongoAsyncRepository) -> None:
    lock = asyncio.Lock()
    tasks = (
        FeedUpdater(
            name=name,
            url=url,
            summary_auth_token=OBSERVER_AUTH_TOKEN,
            repository=repository,
            throttle_lock=lock,
        ).update_feed()
        for name, url in OBSERVER_FEEDS.items()
    )
    await asyncio.gather(*tasks, return_exceptions=True)


async def update_feeds_task() -> None:
    repository = MongoAsyncRepository(
        mongo_uri=OBSERVER_MONGO_URI,
        db_name=OBSERVER_MONGO_DB,
        articles_col_name=OBSERVER_MONGO_ARTICLES,
        feeds_col_name=OBSERVER_MONGO_FEEDS,
    )
    while True:
        await update_feeds(repository)
        await asyncio.sleep(OBSERVER_FEED_UPDATE_TIMEOUT)

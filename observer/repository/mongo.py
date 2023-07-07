import asyncio
from typing import Iterable, Optional

from motor.motor_asyncio import AsyncIOMotorClient, AsyncIOMotorCollection

from models import Article, Feed
from .interface import AsyncRepository


class MongoAsyncRepository(AsyncRepository):
    def __init__(
        self,
        mongo_uri: str,
        db_name: str,
        articles_col_name: str,
        feeds_col_name: str,
        loop: Optional[asyncio.AbstractEventLoop] = None,
    ):
        if loop:
            self._client = AsyncIOMotorClient(host=mongo_uri, io_loop=loop)
        else:
            self._client = AsyncIOMotorClient(host=mongo_uri)
        self._db = self._client[db_name]
        self._articles: AsyncIOMotorCollection = self._db[articles_col_name]
        self._feeds: AsyncIOMotorCollection = self._db[feeds_col_name]

    async def get_articles(self, ids: Iterable[str]) -> Iterable[Article]:
        cursor = self._articles.find({"_id": {"$in": ids}})
        articles = [Article.from_dict(d) async for d in cursor]
        return articles

    async def insert_articles(self, articles: Iterable[Article]) -> None:
        await self._articles.insert_many([a.as_dict() for a in articles])

    async def insert_feed(self, feed: Feed) -> None:
        await self._feeds.update_one(
            {"_id": feed.url}, {"$set": feed.as_dict()}, upsert=True
        )

    async def get_feeds(self, ids: Iterable[str]) -> Iterable[Feed]:
        pipeline = [
            {"$match": {"_id": {"$in": ids}}},
            {"$addFields": {"__order": {"$indexOfArray": [ids, "$_id"]}}},
            {"$sort": {"__order": 1}},
        ]
        res = self._feeds.aggregate(pipeline)
        feeds = [Feed.from_dict(d) async for d in res]
        return feeds

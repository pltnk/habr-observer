import asyncio

from fetcher import update_feeds_task

if __name__ == "__main__":
    asyncio.run(update_feeds_task())

import asyncio

import streamlit as st

from config import (
    OBSERVER_FEEDS,
    OBSERVER_MONGO_URI,
    OBSERVER_MONGO_DB,
    OBSERVER_MONGO_ARTICLES,
    OBSERVER_MONGO_FEEDS,
    OBSERVER_FEED_CACHE_TTL,
)
from repository import MongoAsyncRepository
from .render import render_header, render_toggle, render_tabs, render_footer


@st.cache_resource
def get_event_loop() -> asyncio.AbstractEventLoop:
    return asyncio.new_event_loop()


@st.cache_resource
def create_repository():
    return MongoAsyncRepository(
        mongo_uri=OBSERVER_MONGO_URI,
        db_name=OBSERVER_MONGO_DB,
        articles_col_name=OBSERVER_MONGO_ARTICLES,
        feeds_col_name=OBSERVER_MONGO_FEEDS,
        loop=get_event_loop(),
    )


@st.cache_data(ttl=OBSERVER_FEED_CACHE_TTL)
def get_feeds_sync():
    return get_event_loop().run_until_complete(
        create_repository().get_feeds(list(OBSERVER_FEEDS.values())),
    )


def run_app():
    render_header()
    with st.spinner(text="Читаю статьи..."):
        feeds = get_feeds_sync()
    if feeds:
        collapse_summaries = render_toggle()
        render_tabs(feeds, collapse_summaries=collapse_summaries)
    else:
        st.info("Лента пересобирается, загляните позже 😉")
    render_footer()

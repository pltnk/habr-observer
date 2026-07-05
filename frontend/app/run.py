import httpx
import streamlit as st
from config import OBSERVER_BACKEND_URL, OBSERVER_FEED_CACHE_TTL
from models import Feed

from .render import render_footer, render_header, render_tabs, render_toggle


@st.cache_data(ttl=OBSERVER_FEED_CACHE_TTL)
def get_feeds():
    """Fetch every feed from the backend, or [] when it is unreachable."""
    try:
        response = httpx.get(f"{OBSERVER_BACKEND_URL}/feeds")
        response.raise_for_status()
        return [Feed.from_dict(d) for d in response.json()]
    except httpx.HTTPError:
        return []


def run_app():
    render_header()
    with st.spinner(text="Читаю статьи..."):
        feeds = get_feeds()
    if feeds:
        collapse_summaries = render_toggle()
        render_tabs(feeds, collapse_summaries=collapse_summaries)
    else:
        st.info("Лента пересобирается, загляните позже 😉")
    render_footer()

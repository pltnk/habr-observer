FROM python:3.9-slim
WORKDIR /opt/observer
ENV PYTHONPATH "/opt/observer:$PYTHONPATH"
ENV PYTHONUNBUFFERED true
COPY requirements.txt /opt/observer/requirements.txt
RUN pip install --no-cache-dir --ignore-installed -r requirements.txt
COPY ./app/static/* /usr/local/lib/python3.9/site-packages/streamlit/static/
COPY . /opt/observer
RUN apt update && apt install -y curl \
    && rm -rf /var/lib/apt/lists/*
ARG SL_PORT=8501
EXPOSE ${SL_PORT}
ENV OBSERVER_SL_PORT=${SL_PORT}
HEALTHCHECK CMD curl --fail http://localhost:${OBSERVER_SL_PORT}/_stcore/health
CMD python3 run_fetcher.py & streamlit run run_app.py --server.address=0.0.0.0 --server.port=${OBSERVER_SL_PORT} --browser.gatherUsageStats=false

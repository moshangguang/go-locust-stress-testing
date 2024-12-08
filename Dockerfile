FROM golang:1.22.10 AS golang-builder
LABEL authors="lf"

WORKDIR /app
COPY . .

RUN ["/bin/bash","build.sh"]

FROM python:3.11.11
RUN apt-get update && \
    apt-get install -y cron supervisor && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=golang-builder /app/main .
COPY --from=golang-builder /app/master.py .
COPY --from=golang-builder /app/requirements.txt .
COPY --from=golang-builder /app/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
RUN pip install --upgrade pip
RUN pip install -r /app/requirements.txt

CMD ["/usr/bin/supervisord"]

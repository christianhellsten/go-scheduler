FROM golang:alpine

ENV SIDEKIQ_WORKER
ENV DB_TABLE
ENV REDIS_SERVER
ENV REDIS_DB
ENV REDIS_POOL
ENV REDIS_PROCESS
ENV POLL_INTERVAL

# Install dependencies
RUN apk --no-cache add git

# Build and install from source
RUN go get github.com/christianhellsten/go-scheduler

# Start go-scheduler
ENTRYPOINT ["go-scheduler"]

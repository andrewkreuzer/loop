FROM golang:alpine as build

WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web .


FROM ubuntu:latest
ENV PORT=8080
ENV URL="0.0.0.0"

RUN apt update && apt upgrade -y

RUN useradd -d /home/web -u 1000 web
USER web

WORKDIR /home/web
COPY --from=build /app/web ./web
COPY ./cluster_config.json .

EXPOSE $PORT
CMD ["./web"]

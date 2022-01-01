FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY *.md ./

RUN go build -o /cdn

EXPOSE 7777

CMD [ "/cdn" ]

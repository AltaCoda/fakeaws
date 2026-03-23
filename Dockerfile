FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /fakeaws ./cmd/fakeaws

FROM alpine:3.21
COPY --from=builder /fakeaws /usr/local/bin/fakeaws
EXPOSE 4579
ENTRYPOINT ["fakeaws"]
CMD ["--port", "4579"]

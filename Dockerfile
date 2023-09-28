FROM golang:alpine as builder

# Install protoc and related stuff
RUN apk add --no-cache ca-certificates

RUN mkdir /app
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
COPY go.mod ./
COPY go.sum ./

#ENV GOPROXY https://goproxy.cn,direct

RUN go mod download
# Copy local code to the container image
COPY . .
COPY .env .

WORKDIR /app/api/proto
RUN mkdir -p "code/go"

WORKDIR /app

# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o general_ledger_golang ./cmd/combined/main.go
RUN go build -o general_ledger_golang ./cmd/combined/main.go

# Run container
FROM golang:alpine

RUN mkdir /app

WORKDIR /app

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app .
ENV DOT_ENV=enable

EXPOSE 3000
# Run the web service on container startup.
CMD ["./general_ledger_golang"]
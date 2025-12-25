FROM golang:alpine AS builder

RUN apk add --no-cache gcc musl-dev git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary with static linking for CGO (sqlite3 needs it)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-extldflags '-static'" -o whatsabladerunner main.go

# Prepare config templates - only git-tracked files
RUN mkdir -p /app/templates && \
    git ls-files config/ | xargs -I {} cp --parents {} /app/templates/

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata sqlite-libs

COPY --from=builder /app/whatsabladerunner /usr/local/bin/whatsabladerunner

COPY --from=builder /app/templates/config /usr/local/share/whatsabladerunner/config-template

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

VOLUME /data
WORKDIR /data

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

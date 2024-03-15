FROM golang:1.19-alpine3.18 AS builder

COPY ${PWD} /app
WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct \
    GO111MODULE=on

# Toggle CGO based on your app requirement. CGO_ENABLED=1 for enabling CGO
RUN CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o /app/appbin *.go
# Use below if using vendor
# RUN CGO_ENABLED=0 go build -mod=vendor -ldflags '-s -w -extldflags "-static"' -o /app/appbin *.go

FROM alpine:3.18

# Following commands are for installing CA certs (for proper functioning of HTTPS and other TLS)
RUN apk --update add ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

ENV TZ=Asia/Shanghai

# Add new user 'appuser'
RUN adduser -D appuser
USER appuser

COPY --from=builder /app /home/appuser/app

WORKDIR /home/appuser/app

# Since running as a non-root user, port bindings < 1024 is not possible
# 8000 for HTTP; 8443 for HTTPS;
EXPOSE 8000
EXPOSE 8443
# api-front
EXPOSE 8080

CMD ["./appbin"]
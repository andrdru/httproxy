FROM golang:1.15-alpine AS build

WORKDIR /go/src/github.com/andrdru/httproxy
COPY . /go/src/github.com/andrdru/httproxy

RUN CGO_ENABLED=0 go build -o /bin/httproxy

FROM alpine
COPY --from=build /bin/httproxy /bin/httproxy

RUN printf "#!/bin/sh \n /bin/httproxy \
--address=\$PROXY_ADDRESS \
--hosts=\$PROXY_HOSTS \
--endpoint=\$PROXY_HEALTHCHECK \
--interval=\$PROXY_HEALTHCHECK_INTERVAL \
--health_timeout=\$PROXY_HEALTHCHECK_TIMEOUT \
--timeout=\$PROXY_TIMEOUT \
" > /entrypoint.sh && chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

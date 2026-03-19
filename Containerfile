FROM alpine:latest

WORKDIR /app

COPY ./.dist/opencode-connect /usr/local/bin/opencode-connect
COPY ./configs/config.example.yaml /app/configs/config.yaml

EXPOSE 8192

ENTRYPOINT ["/usr/local/bin/opencode-connect"]
CMD ["-c", "/app/configs/config.yaml"]

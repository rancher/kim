# syntax = docker/dockerfile:1.0-experimental
FROM alpine
RUN --mount=type=secret,id=mysecret cat /run/secrets/mysecret

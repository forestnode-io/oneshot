FROM golang:1.19-alpine AS build-phase
RUN apk add make upx
WORKDIR /oneshot
COPY . .
RUN GOARCH=amd64 GOOS=linux make oneshot && upx --best --ultra-brute /oneshot/build-output/oneshot

FROM --platform=amd64 scratch
COPY --from=build-phase /oneshot/build-output/oneshot /bin/oneshot
WORKDIR /oneshot
ENV USER oneshot
STOPSIGNAL SIGINT
EXPOSE 8080
ENTRYPOINT [ "oneshot" ]
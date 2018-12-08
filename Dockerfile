from golang:1.11.2 as build
add . /src
run cd /src && CGO_ENABLED=0 GOOS=linux go get -v .

from gcr.io/distroless/base
copy --from=build /go/bin/etcdhcp /etcdhcp
expose 9842
entrypoint ["/etcdhcp"]

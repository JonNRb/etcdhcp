from golang:1.10.3 as build
add . /go/src/go.jonnrb.io/etcdhcp
run cd /go/src/go.jonnrb.io/etcdhcp \
 && CGO_ENABLED=0 GOOS=linux go-wrapper download \
 && CGO_ENABLED=0 GOOS=linux go-wrapper install

from gcr.io/distroless/base
copy --from=build /go/bin/etcdhcp /etcdhcp
expose 9842
entrypoint ["/etcdhcp"]

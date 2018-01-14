from golang:1.9 as build
add . /go/src/github.com/jonnrb/etcdhcp
run cd /go/src/github.com/jonnrb/etcdhcp \
 && CGO_ENABLED=0 GOOS=linux go-wrapper download \
 && CGO_ENABLED=0 GOOS=linux go-wrapper install

from gcr.io/distroless/base
copy --from=build /go/bin/etcdhcp /etcdhcp
expose 9842
entrypoint ["/etcdhcp"]

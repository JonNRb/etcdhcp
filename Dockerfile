from quay.io/jonnrb/go as build
add . /src
run cd /src && CGO_ENABLED=0 GOOS=linux go get -v .

from gcr.io/distroless/static
copy --from=build /go/bin/etcdhcp /etcdhcp
expose 9842
entrypoint ["/etcdhcp"]

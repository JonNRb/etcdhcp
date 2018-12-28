from quay.io/jonnrb/go as build
add . .
run CGO_ENABLED=0 go get -v .

from gcr.io/distroless/static
copy --from=build /go/bin/etcdhcp /etcdhcp
expose 9842
entrypoint ["/etcdhcp"]

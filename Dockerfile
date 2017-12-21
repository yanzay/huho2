FROM golang:alpine
WORKDIR /go/src/github.com/yanzay/huho2/
RUN apk update && apk --no-cache add make git
RUN go get -v -u github.com/gopherjs/gopherjs
RUN go get -v -u github.com/jteeuwen/go-bindata/...
RUN go get -v -u github.com/elazarl/go-bindata-assetfs/...
COPY . .
RUN make

FROM alpine
EXPOSE 8080
RUN apk update && apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/yanzay/huho2/huho .
CMD [ "./huho" ]

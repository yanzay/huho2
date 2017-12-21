build:
	go-bindata-assetfs static/...
	go build -v .
run: build
	./server
dev:
	go-bindata-assetfs -debug static/...
	go build -v .
	./server --log-level trace

build: front
	cd server && go-bindata-assetfs ./static/... && cd ..
	go build -v -o huho ./server

front:
	gopherjs build -v -o ./server/static/client.js ./client

dev: build
	./huho

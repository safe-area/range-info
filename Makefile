.PHONY: build
build:
	docker build -t poncheska/sa-data-getter-v2 -f builds/Dockerfile .
	docker push poncheska/sa-data-getter-v2

.PHONY: run
run:
	go run ./main.go

.PHONY: build
build:
	docker build -t poncheska/sa-data-getter -f builds/Dockerfile .
	docker push poncheska/sa-data-getter

.PHONY: run
run:
	go run ./main.go

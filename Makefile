.PHONY: build
build:
	docker build -t poncheska/sa-test -f builds/Dockerfile .
	docker push poncheska/sa-test

.PHONY: run
run:
	go run ./main.go
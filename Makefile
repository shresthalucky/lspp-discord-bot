run: build
	chmod +x main
	./main

build:
	env go build main.go

up:
	docker build -t froggy-bot .
	docker run -d --name froggy-bot froggy-bot

logs:
	docker logs -f froggy-bot

down:
	docker stop froggy-bot

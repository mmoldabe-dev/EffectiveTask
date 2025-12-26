up:
	docker-compose up -d --build


down:
	docker-compose down

run:
	go run cmd/main.go

swag:
	swag init -g cmd/main.go
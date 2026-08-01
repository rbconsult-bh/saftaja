include .env

DB_URL = postgres://$(DB_USER):$(DB_PASSWORD)@db:5432/$(DB_DATABASE)?sslmode=disable

migrate-up:
	docker compose run --rm migrate -path=/migrations -database="$(DB_URL)" up

migrate-down:
	docker compose run --rm migrate -path=/migrations -database="$(DB_URL)" down 1

migrate-create:
	docker compose run --rm migrate create -ext sql -dir /migrations -seq $(name)

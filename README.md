# pay

## run migrations

docker compose run --rm migrate -path=/migrations -database="postgres://somehuman:password@db:5432/paydb?sslmode=disable" up

## create new migration

docker compose run --rm migrate create -ext sql -dir /migrations -seq init_schema

## rollback

docker compose run --rm migrate -path=/migrations -database="postgres://somehuman:password@db:5432/paydb?sslmode=disable" down 1

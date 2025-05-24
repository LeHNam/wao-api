.PHONY: build start logs-app ssh-app

build:
	docker-compose build
start:
	docker-compose up -d


stop: #down
	@echo "=============cleaning up============="
	docker-compose down
	docker system prune -f
	docker volume prune -f
logs-app:
	docker logs -f wao-api
ssh-app:
	docker exec -it wao-api bash

generate:
	wire ./services/wire

run:
	go run main.go


swagger:
	redocly bundle ./api/open-api.yaml -o bundled.yaml

generate-product-api:
	oapi-codegen --config ./api/product/config.yaml ./api/product/api.yaml

generate-user-api:
	oapi-codegen --config ./api/user/config.yaml ./api/user/api.yaml

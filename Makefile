.PHONY: help
help:
	@echo ""
	@echo "Please use \033[1mmake <target>\033[0m where <target> is one of:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## generate-api: openapi generation
generate-api:
	mkdir -p generated/api
	oapi-codegen -package api api/spec/openapi.yaml > generated/api/api.gen.go

## generate-sql: sqlc generation
generate-sql:
	mkdir -p generated/db
	sqlc generate

## generate: generate all
generate: generate-api generate-sql 
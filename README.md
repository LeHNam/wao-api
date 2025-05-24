# wao-api

A Go-based API service built with modern technologies and best practices.

## Installation

### System Requirements
- Go 1.23 or higher
- Make
- Docker (optional)

### Required Tools Installation

```bash
# Install oapi-codegen
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# Install redocly
npm install -g @redocly/cli
```

## Project Structure

```
.
├── README.md
├── api
│ ├── open-api.yaml
│ ├── product
│ │ ├── api.yaml
│ │ ├── config.yaml
│ │ ├── iml.go
│ │ └── server.go
│ ├── security.yaml
│ └── user
│     ├── api.yaml
│     ├── config.yaml
│     ├── iml.go
│     └── server.go
├── build
│ └── app
├── bundled.yaml
├── config
│ ├── config.go
│ └── development.yaml
├── constant
│ └── constant.go
├── context
│ └── service.go
├── docker-compose.yml
├── go.mod
├── go.sum
├── helpers
│ └── utils
│     └── util.go
├── main.go
├── makefile
├── middlewares
│ ├── auth.go
│ └── cors.go
├── models
│ ├── product_options.go
│ ├── products.go
│ └── user.go
└── services
    ├── database
    │ ├── constant.go
    │ ├── errors.go
    │ ├── postgres.go
    │ ├── repository.go
    │ └── types.go
    ├── i18nService
    │ └── i18n.go
    ├── log
    │ └── zap.go
    ├── server
    │ └── server.go
    └── wire
        ├── wire.go
        └── wire_gen.go
```

## New API Creation Process

1. **Create OpenAPI/Swagger Files**
    - Create a new directory in `api/` for the new module
    - Create `api.yaml` to define API endpoints
    - Create `config.yaml` for oapi-codegen

2. **Generate Code**
   ```bash
   # Generate code from OpenAPI spec
   make generate-user-api
   
   # Bundle OpenAPI docs
   make swagger
   ```

3. **Implement Business Logic**
    - Create handlers in `api/`
    - Implement business logic in `services/`
    - Create models in `models/`
    - Implement repository in `repositories/`

4. **Wire Dependencies**
    - Update wire configuration in `services/wire/`

5. **Run Service**
   ```bash
   make run
   ```

## Makefile Commands

- `make generate`: Generate wire dependencies
- `make run`: Run service
- `make swagger`: Bundle OpenAPI docs
- `make generate-user-api`: Generate code from OpenAPI spec for user API

## Development

1. Clone repository
2. Install dependencies
3. Run `make generate` to generate code
4. Run `make run` to start service

## API Documentation

API documentation can be viewed at `/docs` after running the service.

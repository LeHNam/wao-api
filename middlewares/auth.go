package middlewares

import (
	"context"
	"fmt"
	"github.com/LeHNam/wao-api/helpers/utils"
	"strings"

	serverContext "github.com/LeHNam/wao-api/context"
	"github.com/getkin/kin-openapi/openapi3filter"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
)

func BearerAuthMiddleware(sc *serverContext.ServiceContext, isCheckPermission bool) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		token := input.RequestValidationInput.Request.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			return fmt.Errorf("missing %s header", "X-TOKEN")
		}

		user, err := utils.GetTokenClaims(token)
		if err != nil {
			return fmt.Errorf("missing or invalid authorization token")
		}

		ginmiddleware.GetGinContext(ctx).Set("user", user)
		ginmiddleware.GetGinContext(ctx).Set("token", token)

		return nil
	}
}

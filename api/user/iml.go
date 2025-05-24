package user

import (
	"context"
	"fmt"
	svCtx "github.com/LeHNam/wao-api/context"
	"github.com/LeHNam/wao-api/helpers/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	Token    string    `json:"token"`
}

type UserServer struct {
	sc *svCtx.ServiceContext
}

func NewUserServer(sc *svCtx.ServiceContext) *UserServer {
	return &UserServer{
		sc: sc,
	}
}

// PostLogin handles the login API
func (s *UserServer) PostLogin(ctx context.Context, request PostLoginRequestObject) (PostLoginResponseObject, error) {
	user, err := s.sc.UserRepo.FindOne(ctx, map[string]interface{}{
		"username": request.Body.Username,
	}, []string{})
	if err != nil {
		s.sc.Log.Error("user not found", zap.Error(err))
		return PostLogin401JSONResponse{
			Message: utils.Stp("Invalid username or password"),
		}, nil
	}

	// Validate password (hash comparison can be added here)
	valid := utils.CheckPasswordHash(request.Body.Password, user.Password)
	if !valid {
		fmt.Print("pass found", request.Body.Password, user.Password)

		return PostLogin401JSONResponse{
			Message: utils.Stp("Invalid username or password"),
		}, nil
	}

	// Generate JWT token
	token, err := utils.CreateToken(s.sc.Config.JWT.Secret, jwt.MapClaims{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"name":     user.Name,
		"role":     user.Role,
	})
	if err != nil {
		return PostLogin401JSONResponse{
			Message: utils.Stp("failed to create token"),
		}, nil
	}

	// Update user token in the database
	user.Token = token
	err = s.sc.UserRepo.Update(ctx, user.ID, map[string]interface{}{
		"token": token,
	})
	if err != nil {
		s.sc.Log.Error("failed to save token", zap.Error(err))
		return PostLogin401JSONResponse{
			Message: utils.Stp("Internal server error"),
		}, nil
	}

	return PostLogin200JSONResponse{
		Token: &token,
	}, nil
}

// PostLogout handles the logout API
func (s *UserServer) PostLogout(ctx context.Context, request PostLogoutRequestObject) (PostLogoutResponseObject, error) {
	// Extract token from context (assumes middleware sets it)
	tokenString, exists := ctx.Value("token").(string)
	if !exists || tokenString == "" {
		return PostLogout401JSONResponse{
			Message: utils.Stp("Unauthorized"),
		}, nil
	}

	// Parse the token
	err := s.sc.UserRepo.UpdateFields(ctx, map[string]interface{}{
		"token": tokenString,
	}, map[string]interface{}{
		"token": "",
	})
	if err != nil {
		s.sc.Log.Error("failed to invalidate token", zap.Error(err))
		return PostLogout401JSONResponse{
			Message: utils.Stp("Internal server error"),
		}, nil
	}

	return PostLogout200JSONResponse{
		Message: utils.Stp("Logout successful"),
	}, nil
}

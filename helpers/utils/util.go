package utils

import (
	"context"
	"fmt"
	"github.com/LeHNam/wao-api/config"
	"github.com/LeHNam/wao-api/models"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func Stp(s string) *string {
	return &s
}
func CheckPasswordHash(password string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CreateToken(secret string, c jwt.Claims) (string, error) {
	signingKey := []byte(secret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)

	ss, err := token.SignedString(signingKey)
	if err != nil {
		return "", err
	}
	return ss, nil
}

func ParseToken(tokenString string, key string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(key), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func GetTokenClaims(token string) (*models.User, error) {
	cfg := config.GetConfig()
	jwtSecret := cfg.JWT.Secret
	payload, err := ParseToken(token, jwtSecret)
	if err != nil {
		fmt.Println("Error parsing token", err)
		return nil, err
	}

	user := models.User{
		Name:     payload["name"].(string),
		Username: payload["username"].(string),
		Email:    payload["email"].(string),
		Role:     payload["role"].(string),
	}

	id, err := uuid.Parse(payload["id"].(string))
	if err != nil {
		return nil, fmt.Errorf("Invalid UUID for id: %v", err)
	}
	user.ID = id

	return &user, nil
}

func GetUserFromContext(ctx context.Context) *models.User {
	userValue := ctx.Value("user")
	if userValue == nil {
		return nil
	}

	user, ok := userValue.(*models.User)
	if !ok {
		return nil
	}
	return user
}

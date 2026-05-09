package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	secret []byte
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(secret string) *AuthService {
	return &AuthService{secret: []byte(secret)}
}

func (s *AuthService) Login(req LoginRequest) (LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return LoginResponse{}, errors.New("username or password is empty")
	}

	if req.Username != "admin" || req.Password != "admin123" {
		return LoginResponse{}, errors.New("invalid credentials")
	}

	now := time.Now()
	claims := Claims{
		Username: req.Username,
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "skoll2-main",
			Subject:   req.Username,
			ExpiresAt: jwt.NewNumericDate(now.Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return LoginResponse{}, err
	}

	return LoginResponse{Token: signed}, nil
}

func (s *AuthService) ParseToken(tokenStr string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil {
		return Claims{}, err
	}
	if !token.Valid {
		return Claims{}, errors.New("token is invalid")
	}
	return claims, nil
}

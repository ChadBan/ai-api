package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DefaultExpireDuration Token 默认过期时间（30天）
	DefaultExpireDuration = "720h"
	// JwtSecret JWT 密钥
	JwtSecret = "i9XAQ52RR0RrjWtnUY8KThRx5a8TXitR/4LgcCkH3"
)

// EncodeJwtToken 生成 JWT Token
func EncodeJwtToken(userUniqueId int64) (string, error) {
	now := time.Now()
	expireDuration, err := time.ParseDuration(DefaultExpireDuration)
	if err != nil {
		return "", fmt.Errorf("parse duration error: %w", err)
	}
	expireTime := now.Add(expireDuration)

	tok := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expireTime),
		IssuedAt:  jwt.NewNumericDate(now),
		Subject:   fmt.Sprint(userUniqueId),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tok)
	return token.SignedString([]byte(JwtSecret))
}

// DecodeJwtToken 解析 JWT Token
func DecodeJwtToken(tokenString string) (sub string, err error) {
	token, _ := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JwtSecret), nil
	})

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		sub = claims.Subject
		if claims.ExpiresAt.Unix() < time.Now().Unix() {
			err = errors.New("token has expired")
		}
	} else {
		err = errors.New("invalid token")
	}

	return sub, err
}

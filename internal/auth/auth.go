package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error){
	hash, err := bcrypt.GenerateFromPassword([]byte(password),bcrypt.DefaultCost)
	if err != nil{
		return "", err
	}
	return string(hash), nil
}
func CheckPasswordHash(password, hash string) error{
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	dur, _ := time.ParseDuration("1h")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy", 
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()), 
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(dur)),
		Subject: userID.String(),
	})
	return token.SignedString([]byte(tokenSecret))
}
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil{
		fmt.Printf("Error parsing token: %v", err)
		return uuid.Nil, err
	}
	id, err := token.Claims.GetSubject()
	if err != nil{
		fmt.Printf("Error extracting ID: %v", err)
		return uuid.Nil, err
	}
	UUID, err := uuid.Parse(id)
	if err != nil{
		fmt.Printf("Error parsing ID to UUID: %v", err)
		return uuid.Nil, err
	}
	return UUID, nil
}
func GetBearerToken(headers http.Header) (string, error){
	authHeader := headers.Get("Authorization")
	if authHeader == ""{
		return "", fmt.Errorf("No authorization header found")
	}
	if !strings.Contains(authHeader, "Bearer"){
		return "", fmt.Errorf("Authorization Header does not contain Bearer token")
	}
	tokenString := authHeader[7:]
	return tokenString, nil
}
func MakeRefreshToken() (string, error){
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
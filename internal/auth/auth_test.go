package auth

import (
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)
func TestHashPassword(t *testing.T) {
	pw := "password123"
	hash1, err := HashPassword(pw)
	if err != nil{
		t.Error("Failed to hash")
	}
	hash2, err := HashPassword(pw)
	if err != nil{
		t.Error("Failed to hash")
	}
	if hash1 == hash2{
		t.Error("Hash collision")
	}
	longPw := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, err = HashPassword(longPw)
	if err == nil{
		t.Error("Hash should fail on password longer than 72 bytes")
	}
}
func TestCheckPasswordHash(t *testing.T){
	pw := "password123"
    	hash1, err := HashPassword(pw)
    	if err != nil{
		t.Errorf("Failed to hash: %v", err)
    	}
	hash1Err := CheckPasswordHash(pw, hash1)
	if hash1Err != nil{
		t.Error("Hash 1 failed verification")
	}
    	hash2, err := HashPassword(pw)
    	if err != nil{
		t.Errorf("Failed to hash: %v", err)
    	}
	hash2Err := CheckPasswordHash(pw, hash2)
	if hash2Err != nil{
		t.Error("Hash 2 failed verification")
	}
	err = CheckPasswordHash("notpassword", hash1)
	if err == nil{
		t.Errorf("Hash 1 verified wrong password: %v", err)
	}
}
func TestMakeJWT(t *testing.T){
	id := uuid.New()
	validToken, err := MakeJWT(id, "superSecret")
	if err != nil {
		t.Errorf("Failed to create token: %v", err)
	}
	_, err = jwt.Parse(validToken,func(t *jwt.Token) (any, error) {return []byte("superSecret"), nil})
	if err != nil {
		t.Errorf("Invalid Token: %v", err)
	}
	_, err = jwt.Parse(validToken,func(t *jwt.Token) (any, error) {return []byte("superDuperSecret"), nil})
	if err == nil {
		t.Errorf("Validated invalid secret")
	}
}
func TestGetBearerToken(t *testing.T){
	req, err := http.NewRequest(http.MethodGet,"fake.com",nil)
	if err != nil{
		t.Error("Failed to create request")
	}
	req.Header.Set("Authorization", "Bearer abc")
	token, err := GetBearerToken(req.Header)
	if err != nil{
		t.Error("Failed to get bearer token")
	}
	if token != "abc"{
		t.Errorf("Failed to correctly extract token string, recieved: %v, expected abc", token)
	}
}
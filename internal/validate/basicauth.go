package validate

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func BasicAuthPasswordHash(username, password string) (string, error) {
	if err := Username(username); err != nil {
		return "", err
	}
	if err := Password(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("密码加密失败")
	}
	return string(hash), nil
}

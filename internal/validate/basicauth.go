package validate

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func BasicAuthPasswordHash(username, password string) (string, error) {
	if err := BasicAuthUsername(username); err != nil {
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

// htpasswd usernames are line-oriented and colon-delimited.
func BasicAuthUsername(username string) error {
	if err := Username(username); err != nil { return err }
	if strings.Contains(username, ":") || strings.IndexFunc(username, unicode.IsControl) >= 0 {
		return fmt.Errorf("Basic Auth 用户名不能包含冒号或控制字符")
	}
	return nil
}

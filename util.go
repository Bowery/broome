// Copyright 2013-2014 Bowery, Inc.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/nu7hatch/gouuid"
	"strings"
)

func HashPassword(password, salt string) string {
	hash := hmac.New(sha256.New, []byte(salt))
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}

func HashToken() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	token := strings.ToLower(id.String())
	return strings.TrimSpace(token), nil
}

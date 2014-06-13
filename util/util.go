// Copyright 2013-2014 Bowery, Inc.
package util

import (
	"code.google.com/p/go-uuid/uuid"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// must equal the node implementation:
// > require('crypto').createHmac('sha256', 'hello').update('world').digest('hex')
// 'f1ac9702eb5faf23ca291a4dc46deddeee2a78ccdaf0a412bed7714cfffb1cc4'
func HashPassword(password, salt string) string {
	hash := hmac.New(sha256.New, []byte(salt))
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}

func HashToken() string {
	return uuid.New()
}

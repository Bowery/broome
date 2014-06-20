package util

import "testing"

func TestHashPassword(t *testing.T) {
	if HashPassword("world", "hello") != "f1ac9702eb5faf23ca291a4dc46deddeee2a78ccdaf0a412bed7714cfffb1cc4" {
		t.Error("Password Hashing Utility does not work with original Skylab implementation.")
	}
}

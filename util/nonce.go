package util

import (
	"github.com/google/uuid"
	"strings"
)

func GenerateSessionId() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}

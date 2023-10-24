//go:build faker
// +build faker

package faker

import (
	"math/rand"
	"time"

	"github.com/bxcodec/faker/v3"
)

func MigFakerInit() {
	faker.SetRandomSource(rand.NewSource(time.Now().UnixNano()))
	CustomGenerator()
}

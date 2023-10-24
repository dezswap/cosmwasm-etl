package faker

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/bxcodec/faker/v3"
)

// CustomGenerator ...
func CustomGenerator() {
	point := []string{"", "."}
	_ = faker.AddProvider("amountString", func(v reflect.Value) (interface{}, error) {
		rand.NewSource(time.Now().UnixNano())
		amount := fmt.Sprintf("%d%s%d", rand.Intn(0xFFFFFFFF), point[rand.Intn(2)], rand.Intn(0xFFFFFFFF))
		return amount, nil
	})

	type meta struct {
		Item      map[string]string   `json:"item"`
		ArrayItem map[string][]string `json:"arrayItem"`
		IntItem   map[string]int      `json:"intItem"`
	}

	_ = faker.AddProvider("meta", func(v reflect.Value) (interface{}, error) {
		metaData := meta{}
		if err := faker.FakeData(&metaData); err != nil {
			panic(err)
		}
		smeta := make(map[string]interface{})
		smeta["data"] = metaData
		return smeta, nil
	})
}

func FakeData(target interface{}) error {
	return faker.FakeData(target)
}

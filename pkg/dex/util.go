package dex

import (
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
)

func QueryToBase64Str[T ContractQueryRequest](query T) (string, error) {
	bytes, err := json.Marshal(query)
	if err != nil {
		return "", errors.Wrap(err, "QueryToBase64Str")
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}

func QueryToJsonStr[T ContractQueryRequest](query T) (string, error) {
	bytes, err := json.Marshal(query)
	if err != nil {
		return "", errors.Wrap(err, "QueryToJsonStr")
	}

	return string(bytes), nil
}

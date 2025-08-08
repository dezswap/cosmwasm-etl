package dex

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type comparable interface {
	string | int | Asset
}

func IndexOf[T comparable](slice []T, target T) int {
	for idx, el := range slice {
		if el == target {
			return idx
		}
	}
	return -1
}

func GetAssetsFromAssetsString(amountsAssets string) ([]Asset, error) {
	assets := strings.Split(amountsAssets, ",")
	for i, a := range assets {
		assets[i] = strings.TrimSpace(a)
	}
	if len(assets) != 2 {
		return nil, errors.New(fmt.Sprintf("wrong format of assetsAmount(%s)", amountsAssets))
	}

	res := []Asset{}
	for _, assetStr := range assets {
		asset, err := GetAssetFromAmountAssetString(assetStr)
		if err != nil {
			return nil, errors.Wrap(err, "getAssetsFromAssetsString")
		}
		res = append(res, asset)
	}
	return res, nil
}

func GetAssetFromAmountAssetString(amountAsset string) (Asset, error) {
	amountAsset = strings.TrimSpace(amountAsset)
	regex, _ := regexp.Compile(`\d+`)
	amount := regex.FindString(amountAsset)
	addr := amountAsset[len(amount):]
	if amount == "" || addr == "" {
		return Asset{}, errors.New("string format must be 0000AAAA")
	}
	return Asset{
		Addr:   addr,
		Amount: amount,
	}, nil
}

func AmountAdd(amount1, amount2 string) (string, error) {
	a1, err := ToBigInt(amount1)
	if err != nil {
		return "", errors.Wrap(err, "dex.AmountAdd")
	}
	a2, err := ToBigInt(amount2)
	if err != nil {
		return "", errors.Wrap(err, "dex.AmountAdd")
	}

	return a1.Add(a1, a2).String(), nil
}

func ToBigInt(n string) (*big.Int, error) {
	bi, ok := big.NewInt(0).SetString(n, 10)
	if !ok {
		return nil, errors.New("invalid number")
	}

	return bi, nil
}

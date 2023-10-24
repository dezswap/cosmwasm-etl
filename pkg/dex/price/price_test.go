package price

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculatePrice(t *testing.T) {
	assert := assert.New(t)

	p := &priceImpl{}
	dec, err := p.calculatePrice("2000000000000000000", 18, "1000000", 6, false)

	assert.NoError(err)
	assert.Equal(dec.String(), "0.500000000000000000")
}

func TestCalculatePrice_Negative(t *testing.T) {
	assert := assert.New(t)

	p := &priceImpl{}
	dec, err := p.calculatePrice("-2000000000000000000", 18, "1000000", 6, false)

	assert.NoError(err)
	assert.Equal(dec.String(), "0.500000000000000000")
}

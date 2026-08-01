package money

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var (
	ErrUnsupportedCurrency = errors.New("unsupported currency")
	ErrInvalidDecimal      = errors.New("invalid decimal amount")
	ErrInvalidPrecision    = errors.New("amount has more precision than currency supports")
	ErrAmountOutOfRange    = errors.New("amount is outside the supported range")
)

type MinorAmount int64

type Currency string

const CurrencyBHD Currency = "BHD"

type currencyMetadata struct {
	exponent int32
}

var supportedCurrencies = map[Currency]currencyMetadata{
	CurrencyBHD: {exponent: 3},
}

func (c Currency) Validate() error {
	_, err := c.Exponent()
	return err
}

func (c Currency) Exponent() (int32, error) {
	metadata, ok := supportedCurrencies[c]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedCurrency, c)
	}

	return metadata.exponent, nil
}

func ParseDecimalString(value string, currency Currency) (MinorAmount, error) {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidDecimal, value)
	}

	exponent, err := currency.Exponent()
	if err != nil {
		return 0, err
	}

	scaled := amount.Shift(exponent)
	if !scaled.Equal(scaled.Truncate(0)) {
		return 0, fmt.Errorf("%w: %s %s", ErrInvalidPrecision, amount.String(), currency)
	}

	integer := scaled.BigInt()
	if !integer.IsInt64() {
		return 0, fmt.Errorf("%w: %s %s", ErrAmountOutOfRange, amount.String(), currency)
	}

	return MinorAmount(integer.Int64()), nil
}

func (a MinorAmount) decimal(currency Currency) (decimal.Decimal, error) {
	exponent, err := currency.Exponent()
	if err != nil {
		return decimal.Decimal{}, err
	}

	return decimal.NewFromInt(int64(a)).Shift(-exponent), nil
}

func (a MinorAmount) DecimalString(currency Currency) (string, error) {
	amount, err := a.decimal(currency)
	if err != nil {
		return "", err
	}

	return amount.String(), nil
}

func (a MinorAmount) Format(currency Currency) (string, error) {
	exponent, err := currency.Exponent()
	if err != nil {
		return "", err
	}

	amount, err := a.decimal(currency)
	if err != nil {
		return "", err
	}
	return amount.StringFixed(exponent), nil
}

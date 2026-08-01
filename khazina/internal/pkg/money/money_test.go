package money

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrencyExponent(t *testing.T) {
	exponent, err := CurrencyBHD.Exponent()
	require.NoError(t, err)
	assert.Equal(t, int32(3), exponent)

	_, err = Currency("USD").Exponent()
	assert.ErrorIs(t, err, ErrUnsupportedCurrency)
}

func TestParseDecimalString(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		expected MinorAmount
	}{
		{name: "whole_amount", amount: "15", expected: 15000},
		{name: "three_decimal_places", amount: "1.250", expected: 1250},
		{name: "smallest_bhd_unit", amount: "0.001", expected: 1},
		{name: "negative_amount", amount: "-1.250", expected: -1250},
		{name: "maximum_minor_amount", amount: "9223372036854775.807", expected: MinorAmount(math.MaxInt64)},
		{name: "minimum_minor_amount", amount: "-9223372036854775.808", expected: MinorAmount(math.MinInt64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := ParseDecimalString(tt.amount, CurrencyBHD)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, amount)
		})
	}
}

func TestParseDecimalStringRejectsUnsupportedValues(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		currency Currency
		err      error
	}{
		{name: "excess_precision", amount: "1.0001", currency: CurrencyBHD, err: ErrInvalidPrecision},
		{name: "positive_overflow", amount: "9223372036854775.808", currency: CurrencyBHD, err: ErrAmountOutOfRange},
		{name: "negative_overflow", amount: "-9223372036854775.809", currency: CurrencyBHD, err: ErrAmountOutOfRange},
		{name: "invalid_decimal", amount: "not-money", currency: CurrencyBHD, err: ErrInvalidDecimal},
		{name: "unsupported_currency", amount: "1", currency: Currency("USD"), err: ErrUnsupportedCurrency},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDecimalString(tt.amount, tt.currency)
			assert.True(t, errors.Is(err, tt.err))
		})
	}
}

func TestMinorAmountDecimalString(t *testing.T) {
	tests := []struct {
		name     string
		amount   MinorAmount
		expected string
	}{
		{name: "whole_amount", amount: 15000, expected: "15"},
		{name: "meaningful_fraction", amount: 15001, expected: "15.001"},
		{name: "trailing_fractional_zeroes", amount: 15100, expected: "15.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted, err := tt.amount.DecimalString(CurrencyBHD)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, formatted)
		})
	}

	_, err := MinorAmount(15000).DecimalString(Currency("USD"))
	assert.ErrorIs(t, err, ErrUnsupportedCurrency)
}

func TestMinorAmountFormat(t *testing.T) {
	formatted, err := MinorAmount(15000).Format(CurrencyBHD)
	require.NoError(t, err)
	assert.Equal(t, "15.000", formatted)

	formatted, err = MinorAmount(1).Format(CurrencyBHD)
	require.NoError(t, err)
	assert.Equal(t, "0.001", formatted)

	_, err = MinorAmount(15000).Format(Currency("USD"))
	assert.ErrorIs(t, err, ErrUnsupportedCurrency)
}

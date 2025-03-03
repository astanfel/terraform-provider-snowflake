package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDataType(t *testing.T) {
	t.Run("with valid data type", func(t *testing.T) {
		ok := IsValidDataType("VARCHAR")
		assert.Equal(t, ok, true)
	})

	t.Run("with invalid data type", func(t *testing.T) {
		ok := IsValidDataType("foo")
		assert.Equal(t, ok, false)
	})
}

func TestValidObjectidentifier(t *testing.T) {
	t.Run("with valid object identifier", func(t *testing.T) {
		ok := validObjectidentifier(randomAccountObjectIdentifier(t))
		assert.Equal(t, ok, true)
	})

	t.Run("with invalid object identifier", func(t *testing.T) {
		ok := validObjectidentifier(NewAccountObjectIdentifier(""))
		assert.Equal(t, ok, false)
	})

	t.Run("over 255 characters", func(t *testing.T) {
		ok := validObjectidentifier(NewAccountObjectIdentifier(randomStringRange(t, 256, 256)))
		assert.Equal(t, ok, false)
	})
}

func TestAnyValueSet(t *testing.T) {
	t.Run("with one value set", func(t *testing.T) {
		ok := anyValueSet(String("foo"))
		assert.Equal(t, ok, true)
	})

	t.Run("with no values", func(t *testing.T) {
		ok := anyValueSet()
		assert.Equal(t, ok, false)
	})

	t.Run("with multiple values set", func(t *testing.T) {
		ok := anyValueSet(String("foo"), String("bar"))
		assert.Equal(t, ok, true)
	})

	t.Run("with multiple values set and nil", func(t *testing.T) {
		ok := anyValueSet(String("foo"), nil, String("bar"))
		assert.Equal(t, ok, true)
	})
}

func TestExactlyOneValueSet(t *testing.T) {
	t.Run("with one value set", func(t *testing.T) {
		ok := exactlyOneValueSet(String("foo"))
		assert.Equal(t, ok, true)
	})

	t.Run("with no values", func(t *testing.T) {
		ok := exactlyOneValueSet()
		assert.Equal(t, ok, false)
	})

	t.Run("with multiple values set", func(t *testing.T) {
		ok := exactlyOneValueSet(String("foo"), String("bar"))
		assert.Equal(t, ok, false)
	})

	t.Run("with multiple values set and nil", func(t *testing.T) {
		ok := exactlyOneValueSet(String("foo"), nil, String("bar"))
		assert.Equal(t, ok, false)
	})
}

func TestEveryValueSet(t *testing.T) {
	t.Run("with one value set", func(t *testing.T) {
		ok := everyValueSet(String("foo"))
		assert.Equal(t, ok, true)
	})

	t.Run("with no values", func(t *testing.T) {
		ok := everyValueSet()
		assert.Equal(t, ok, true)
	})

	t.Run("with multiple values set", func(t *testing.T) {
		ok := everyValueSet(String("foo"), String("bar"))
		assert.Equal(t, ok, true)
	})

	t.Run("with multiple values set and nil", func(t *testing.T) {
		ok := everyValueSet(String("foo"), nil, String("bar"))
		assert.Equal(t, ok, false)
	})
}

func TestEveryValueNil(t *testing.T) {
	t.Run("with one value set", func(t *testing.T) {
		ok := everyValueNil(String("foo"))
		assert.Equal(t, ok, false)
	})

	t.Run("with no values", func(t *testing.T) {
		ok := everyValueNil()
		assert.Equal(t, ok, true)
	})

	t.Run("with multiple values set", func(t *testing.T) {
		ok := everyValueNil(String("foo"), String("bar"))
		assert.Equal(t, ok, false)
	})

	t.Run("with multiple values set and nil", func(t *testing.T) {
		ok := everyValueNil(String("foo"), nil, String("bar"))
		assert.Equal(t, ok, false)
	})
}

func TestValueSet(t *testing.T) {
	t.Run("with value set", func(t *testing.T) {
		ok := valueSet(String("foo"))
		assert.Equal(t, ok, true)
	})

	t.Run("with no value", func(t *testing.T) {
		ok := valueSet(nil)
		assert.Equal(t, ok, false)
	})

	t.Run("with valid identifier", func(t *testing.T) {
		ok := valueSet(NewAccountObjectIdentifier("foo"))
		assert.Equal(t, ok, true)
	})

	t.Run("with invalid identifier", func(t *testing.T) {
		ok := valueSet(NewAccountObjectIdentifier(""))
		assert.Equal(t, ok, false)
	})
}

func TestValidateIntInRange(t *testing.T) {
	t.Run("with value in range", func(t *testing.T) {
		ok := validateIntInRange(5, 0, 10)
		assert.Equal(t, ok, true)
	})

	t.Run("with value out of range", func(t *testing.T) {
		ok := validateIntInRange(5, 10, 20)
		assert.Equal(t, ok, false)
	})
}

func TestValidateIntGreaterThanOrEqual(t *testing.T) {
	t.Run("with value in range", func(t *testing.T) {
		ok := validateIntGreaterThanOrEqual(5, 0)
		assert.Equal(t, ok, true)
	})

	t.Run("with value out of range", func(t *testing.T) {
		ok := validateIntGreaterThanOrEqual(5, 10)
		assert.Equal(t, ok, false)
	})
}

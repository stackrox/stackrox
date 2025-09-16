package sliceutils

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConvertSlice_IntToString tests conversion from int slice to string slice
func TestConvertSlice_IntToString(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}

	// Conversion function: convert int to string
	convertFunc := func(i int) string {
		return strconv.Itoa(i)
	}

	expected := []string{"1", "2", "3", "4", "5"}
	result := ConvertSlice(input, convertFunc)
	assert.Equal(t, expected, result)
}

// TestConvertSlice_EmptyInput tests conversion on an empty slice
func TestConvertSlice_EmptyInput(t *testing.T) {
	input := []int{}

	// Conversion function: convert int to string
	convertFunc := func(i int) string {
		return strconv.Itoa(i)
	}

	expected := []string{}
	result := ConvertSlice(input, convertFunc)
	assert.Equal(t, expected, result)
}

// TestConvertSlice_Nil tests conversion on a nil slice
func TestConvertSlice_Nil(t *testing.T) {
	var input []int

	// Conversion function: convert int to string
	convertFunc := func(i int) string {
		return strconv.Itoa(i)
	}

	result := ConvertSlice(input, convertFunc)
	assert.Nil(t, result, "The conversion of a nil slice should return nil.")
}

// TestConvertSlice_StructToString tests converting a slice of structs to strings
func TestConvertSlice_StructToString(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	input := []Person{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	// Conversion function: convert Person to string
	convertFunc := func(p Person) string {
		return fmt.Sprintf("%s (%d)", p.Name, p.Age)
	}
	expected := []string{"Alice (25)", "Bob (30)", "Charlie (35)"}
	result := ConvertSlice(input, convertFunc)
	assert.Equal(t, expected, result)
}

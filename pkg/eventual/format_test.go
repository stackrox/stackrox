package eventual

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestValue_Format(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var v Value[string]
		for format, expected := range map[string]string{
			"%s":  "<nil>",
			"%q":  `"<nil>"`,
			"%v":  "<nil>",
			"%#v": `(eventual.Value[string])(nil)`,
			"%+v": "<nil>",
			"%T":  "*eventual.value[string]",
			"%p":  "0x0",
		} {
			assert.Equal(t, expected, fmt.Sprintf(format, v), format)
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Run("unset", func(t *testing.T) {
			v := New[string]()
			for format, expected := range map[string]string{
				"%s":  "<unset>",
				"%q":  `"<unset>"`,
				"%v":  "<unset>",
				"%+v": "<unset>",
				"%#v": `(eventual.Value[string]){current:<unset> default:""}`,
				"%d":  "%!d(string=<unset>)",
				"%T":  "*eventual.value[string]",
				"%p":  fmt.Sprintf("%v", unsafe.Pointer(v)), //#nosec G103
			} {
				assert.Equal(t, expected, fmt.Sprintf(format, v), format)
			}
		})

		t.Run("string set", func(t *testing.T) {
			v := New(WithDefaultValue("value"))
			for format, expected := range map[string]string{
				"%s":  "value",
				"%q":  `"value"`,
				"%v":  "value",
				"%#v": `(eventual.Value[string]){current:"value" default:"value"}`,
				"%+v": "value",
				"%T":  "*eventual.value[string]",
				"%d":  "%!d(string=value)",
				"%p":  fmt.Sprintf("%v", unsafe.Pointer(v)), //#nosec G103
			} {
				assert.Equal(t, expected, fmt.Sprintf(format, v), format)
			}
		})
	})

	t.Run("bool", func(t *testing.T) {
		t.Run("unset", func(t *testing.T) {
			v := New[bool]()
			for format, expected := range map[string]string{
				"%s":  "<unset>",
				"%q":  `"<unset>"`,
				"%v":  "<unset>",
				"%+v": "<unset>",
				"%#v": `(eventual.Value[bool]){current:<unset> default:false}`,
				"%T":  "*eventual.value[bool]",
				"%p":  fmt.Sprintf("%v", unsafe.Pointer(v)), //#nosec G103
			} {
				assert.Equal(t, expected, fmt.Sprintf(format, v), format)
			}
		})

		t.Run("bool set", func(t *testing.T) {
			v := New(WithDefaultValue(true))
			for format, expected := range map[string]string{
				"%s":  "%!s(bool=true)",
				"%q":  "%!q(bool=true)",
				"%t":  "true",
				"%v":  "true",
				"%#v": "(eventual.Value[bool]){current:true default:true}",
				"%+v": "true",
				"%T":  "*eventual.value[bool]",
				"%p":  fmt.Sprintf("%v", unsafe.Pointer(v)), //#nosec G103
			} {
				assert.Equal(t, expected, fmt.Sprintf(format, v), format)
			}
		})
	})

	t.Run("sign, width, precision", func(t *testing.T) {
		v := New(WithDefaultValue(55.55))

		assert.Equal(t, "==+0000055.5==",
			fmt.Sprintf("==%+010.1f==", v))
		assert.Equal(t, "==        (eventual.Value[float64]){current:55.55 default:55.55}==",
			fmt.Sprintf("==%#62v==", v))
	})

	t.Run("struct", func(t *testing.T) {

		type testStruct struct {
			s string
			n int
		}

		var v Value[testStruct]
		assert.Equal(t, "<nil>", fmt.Sprint(v))
		assert.Equal(t, `(eventual.Value[eventual.testStruct])(nil)`,
			fmt.Sprintf("%#v", v))

		v = New[testStruct]()
		assert.Equal(t, "<unset>", fmt.Sprint(v))
		assert.Equal(t, `(eventual.Value[eventual.testStruct]){current:<unset> default:eventual.testStruct{s:"", n:0}}`,
			fmt.Sprintf("%#v", v))

		v.Set(testStruct{"abc", 42})
		assert.Equal(t, "{abc 42}", fmt.Sprint(v))
		assert.Equal(t, `(eventual.Value[eventual.testStruct]){current:eventual.testStruct{s:"abc", n:42} default:eventual.testStruct{s:"", n:0}}`,
			fmt.Sprintf("%#v", v))
	})
}

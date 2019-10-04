package debug

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func foo() string {
	return "foo"
}

func bar(x int) (string, error) {
	return fmt.Sprintf("bar: %d", x), nil
}

func baz(x int) int {
	return 2 * x
}

func twoReturnArgs(x int) (string, string) {
	return strconv.FormatInt(int64(x), 10), strconv.FormatInt(int64(x), 16)
}

func varArgsJoiner(args ...string) string {
	return strings.Join(args, "/")
}

func TestLazyCall_Success(t *testing.T) {
	result := fmt.Sprint(LazyCall(foo), LazyCall(bar, 5))
	assert.Equal(t, "foo bar: 5, <nil>", result)
}

func TestLazyCall_SuccessNested(t *testing.T) {
	result := fmt.Sprint(LazyCall(bar, LazyCall(baz, 10)))
	assert.Equal(t, "bar: 20, <nil>", result)
}

func TestLazyCall_SuccessVarArgs(t *testing.T) {
	result := fmt.Sprint(LazyCall(varArgsJoiner, "foo", LazyCall(twoReturnArgs, 10)))
	assert.Equal(t, "foo/10/a", result)
}

func TestLazyCall_ErrorWrongArgCount(t *testing.T) {
	result := fmt.Sprint(LazyCall(baz))
	assert.True(t, strings.HasPrefix(result, "<!"))
}

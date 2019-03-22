package httputil

import (
	"io/ioutil"
	"net/http"
)

// ReadResponse reads the value from the response and closes the body
func ReadResponse(response *http.Response) ([]byte, error) {
	defer func() {
		_ = response.Body.Close()
	}()
	return ioutil.ReadAll(response.Body)
}

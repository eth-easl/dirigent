package request_persistence

import (
	"cluster_manager/internal/data_plane/proxy/requests"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestWrapperSerialization(t *testing.T) {
	request, _ := http.NewRequest("GET", "http://google.com/", strings.NewReader("Hello, World!"))

	w := CreateWrapper(&requests.BufferedRequest{
		Request: *request,
	})

	val, err := json.Marshal(w)
	if err != nil {
		// test will fail if the struct is not marshallable
		t.Error(err)
	}

	var val2 BufferedRequestWrapper
	err = json.Unmarshal(val, &val2)
	if err != nil {
		// test will fail if the struct is not unmarshallable
		t.Error(err)
	}

	if val2.Body != w.Body {
		t.Error("Invalid body in unmarshalled data.")
	}
}

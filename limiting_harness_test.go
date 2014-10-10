package dangerroom

import (
	"io/ioutil"
	"strings"
	"testing"
)

func TestLimit(t *testing.T) {
	h := &LimitingHarness{1}

	ctx := Setup(t, h)
	defer ctx.Close()

	req, err := ctx.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := ctx.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Status code not 200: %d", res.StatusCode)
	}

	by, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err == nil {
		t.Fatal("Expected EOF not found")
	}

	if e := err.Error(); !strings.Contains(e, "unexpected EOF") {
		t.Errorf("Expected EOF error, got: %s", e)
	}

	if body := string(by); body != "o" {
		t.Errorf("Body not 'o': '%s'", body)
	}
}

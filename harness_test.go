package dangerroom

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNoOp(t *testing.T) {
	ctx := Setup(t, nil)
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
	if err != nil {
		t.Fatal(err)
	}

	if body := string(by); body != "ok" {
		t.Errorf("Body not 'ok': '%s'", body)
	}
}

func Setup(t *testing.T, h Harness) *TestContext {
	http.DefaultServeMux.HandleFunc("/origin/", func(w http.ResponseWriter, r *http.Request) {
		head := w.Header()
		head.Set("Content-Type", "text/plain")
		head.Set("Content-Length", "2")
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	srv := httptest.NewServer(http.DefaultServeMux)
	ctx := &TestContext{srv, t}

	u, err := url.Parse(srv.URL + "/origin")
	if err != nil {
		t.Fatal(err.Error())
	}

	proxy := NewSingleHostProxy(u, h, nil)
	http.DefaultServeMux.Handle("/proxy", proxy)

	return ctx
}

type TestContext struct {
	Server *httptest.Server
	*testing.T
}

func (t *TestContext) Do(r *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(r)
}

func (t *TestContext) NewRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	proxiedUrl := t.Server.URL + "/proxy" + urlStr
	return http.NewRequest(method, proxiedUrl, body)
}

func (t *TestContext) Close() {
	t.Server.Close()
}

package dangerroom

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"regexp"

	"fmt"
)

type Server struct {
	mux *http.ServeMux
	*http.Server
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	srv := &Server{mux, &http.Server{Addr: addr, Handler: mux}}
	setupHandler(mux, "/~danger/")
	return srv
}

func setupHandler(mux *http.ServeMux, path string) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		resource, err := getResource(r)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error() + "\n"))
			return
		}

		if resource == nil {
			w.WriteHeader(404)
			w.Write([]byte("No harness for the given Content-Type\n"))
			return
		}

		setResource(w, r, mux, resource)
	})
}

func setResource(w http.ResponseWriter, r *http.Request, mux *http.ServeMux, resource HarnessResource) {
	target, harness := resource.TargetAndHarness()

	u, err := url.Parse(target)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error() + "\n"))
		return
	}

	var prefix string
	if len(r.URL.Path) <= dangerPrefixLen {
		prefix = "/proxy"
	} else {
		prefix = r.URL.Path[dangerPrefixLen-1:]
	}

	if proxy, ok := proxiedPaths[prefix]; ok {
		proxy.Harness = harness
	} else {
		proxy = NewSingleHostProxy(u, prefix, harness, nil)
		proxiedPaths[prefix] = proxy
		mux.Handle(prefix+"/", proxy)
	}

	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("%s resource set on %s\n", reflect.TypeOf(resource), prefix)))
}

func getResource(r *http.Request) (HarnessResource, error) {
	ctype := r.Header.Get("Content-Type")
	matches := ctypeRE.FindStringSubmatch(ctype)
	if len(matches) < 2 {
		return nil, nil
	}

	if resourceFunc, ok := Harnesses[matches[1]]; ok {
		resource := resourceFunc()
		err := json.NewDecoder(r.Body).Decode(resource)
		r.Body.Close()
		return resource, err
	}

	return nil, nil
}

var (
	ctypeRE         = regexp.MustCompile(`\Aapplication\/vnd\.danger\-room\.([\w\-\_]+)\+json`)
	dangerPrefixLen = len("/~danger/")
	proxiedPaths    = make(map[string]*Proxy)
)

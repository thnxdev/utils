package httpgh

import "net/http"

type Transport struct {
	T http.RoundTripper
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Accept", "application/vnd.github.hawkgirl-preview+json")
	return t.T.RoundTrip(req)
}

func NewTransport(T http.RoundTripper) *Transport {
	if T == nil {
		T = http.DefaultTransport
	}
	return &Transport{T}
}

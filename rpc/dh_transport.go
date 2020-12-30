package rpc

import (
	"net/http"
)

type DHTransport struct {
	rt  http.RoundTripper
	key string
}

func (dht DHTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Add("Authorization", dht.key)
	return dht.rt.RoundTrip(r)
}

func NewDHTransport(key string) DHTransport {
	return DHTransport{
		rt:  http.DefaultTransport,
		key: key,
	}
}

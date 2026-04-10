package yagpt

import "net/http"

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

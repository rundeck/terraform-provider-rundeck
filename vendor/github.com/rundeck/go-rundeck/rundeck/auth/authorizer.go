package auth

import "net/http"

import "github.com/Azure/go-autorest/autorest"

type TokenAuthorizer struct {
	Token string
}

func (ta *TokenAuthorizer) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err == nil {
				r.Header.Set("X-Rundeck-Auth-Token", ta.Token)

				accept := r.Header.Get("Accept")

				if accept == "" {
					r.Header.Set("Accept", "application/json")
				}
			}
			return r, err
		})
	}
}

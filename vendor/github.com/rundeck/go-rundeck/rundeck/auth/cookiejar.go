package auth

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

// RundeckCookieJar overrides SetCookies to prevent JSESSIONID from being set
type RundeckCookieJar struct {
	*cookiejar.Jar
}

func (j *RundeckCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	var filteredCookies []*http.Cookie

	for _, cookie := range cookies {
		if cookie.Name != "JSESSIONID" {
			filteredCookies = append(filteredCookies, cookie)
		}
	}

	j.Jar.SetCookies(u, filteredCookies)
}

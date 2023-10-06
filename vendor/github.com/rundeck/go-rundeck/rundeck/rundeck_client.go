package rundeck

import (
	"net/http"
	"net/http/cookiejar"

	"github.com/rundeck/go-rundeck/rundeck/auth"
)

func setSender(client BaseClient) BaseClient {

	jar, _ := cookiejar.New(nil)

	j := &auth.RundeckCookieJar{
		Jar: jar,
	}

	client.Sender = &http.Client{Jar: j}
	return client
}

// NewRundeckWithBaseURI creates an instance of the Rundeck client.
func NewRundeckWithBaseURI(baseURI string) BaseClient {
	client := NewWithBaseURI(baseURI)

	return setSender(client)
}

package rundeck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type TokenResp struct {
	User       string    `json:"user"`
	Token      string    `json:"token"`
	ID         string    `json:"id"`
	Creator    string    `json:"creator"`
	Name       string    `json:"name"`
	Expiration time.Time `json:"expiration"`
	Roles      []string  `json:"roles"`
	Expired    bool      `json:"expired"`
}

func getToken(d *schema.ResourceData) (string, error) {
	urlP, _ := d.Get("url").(string)
	apiVersion, _ := d.Get("api_version").(string)
	username, _ := d.Get("auth_username").(string)
	password, _ := d.Get("auth_password").(string)

	return _getToken(urlP, apiVersion, username, password)

}

func _getToken(urlP string, apiVersion string, username string, password string) (string, error) {

	secCheckUrlString := fmt.Sprintf("%s/j_security_check", urlP)
	secCheckUrl, err := url.Parse(secCheckUrlString)

	if err != nil {
		return "", err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Jar: jar,
	}

	data := url.Values{
		"j_username": {username},
		"j_password": {password},
	}
	_, err = client.PostForm(secCheckUrl.String(), data)
	if err != nil {
		return "", err
	}

	tokenUrlString := fmt.Sprintf("%s/api/%s/tokens", urlP, "43")
	tokenUrl, err := url.Parse(tokenUrlString)

	if err != nil {
		return "", err
	}

	tokenBody := map[string]interface{}{}
	tokenBody["user"] = username
	tokenBody["roles"] = []string{"*"}
	tokenBody["duration"] = "0"
	tokenBody["name"] = "terraform-token"
	tokenBodyJson, err := json.Marshal(tokenBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", tokenUrl.String(), bytes.NewBuffer(tokenBodyJson))
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("statuscode %d\n%s", resp.StatusCode, string(body))
	} else if err != nil {
		return "", err
	}

	tokenResp := &TokenResp{}
	err = json.NewDecoder(resp.Body).Decode(tokenResp)
	if err != nil {
		return "", err
	}

	if version, _ := strconv.Atoi(apiVersion); version >= 19 {
		return tokenResp.Token, nil
	} else if tokenResp.Token != "" {
		return tokenResp.Token, nil
	} else if tokenResp.ID != "" {
		return tokenResp.ID, nil
	}

	return tokenResp.Token, nil

}

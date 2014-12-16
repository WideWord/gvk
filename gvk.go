package gvk

import(
	"errors"
	"net/http"
	"net/url"
	"io/ioutil"
	"encoding/json"
	"time"
	"fmt"

)

type ServerSession struct {
	appID string
	appSecret string
	accessToken string
	CallDelay time.Duration
	callDelayTimer <-chan time.Time
}

func Server(appID string, appSecret string) (result *ServerSession, err error) {

	session := &ServerSession{}
	session.CallDelay = 0
	session.callDelayTimer = time.After(0)
	session.appID = appID
	session.appSecret = appSecret

	query, err := url.Parse("https://oauth.vk.com/access_token")
	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("v", "5.24")
	params.Set("grant_type", "client_credentials")
	query.RawQuery = params.Encode()

	url := query.String()

	resp, err := http.Get(url)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var parsedData struct {
		Error string
		Access_token string
	}

	err = json.Unmarshal(data, &parsedData)
	if err != nil {
		return
	}

	if parsedData.Error != "" {
		err = errors.New(parsedData.Error)
	}

	session.accessToken = parsedData.Access_token

	result = session
	err = nil
	return
}

func (session *ServerSession) PlainCall(method string, params url.Values, response interface{}) (err error) {

	query, err := url.Parse("https://api.vk.com/")

	query.Scheme = "https"
	query.Host = "api.vk.com"
	query.Path = fmt.Sprintf("/method/%s", method)
	query.RawQuery = params.Encode()

	url := query.String()

	resp, err := http.Get(url)

	if err != nil { panic(err) }

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil { panic(err) }

	var parsedData struct {
		Error struct {
			Error_code int
			Error_msg string
		}
		Response interface{}
	}

	parsedData.Response = response

	err = json.Unmarshal(data, &parsedData)
	if err != nil {
		err = errors.New(fmt.Sprintf("%s\n====\n%s\n====", err.Error(), data))
		return
	}

	if parsedData.Error.Error_msg != "" { err = errors.New(parsedData.Error.Error_msg); return }

	return nil
}


func (session *ServerSession) AuthCall(method string, params url.Values, response interface{}) error {
	<- session.callDelayTimer
	params.Add("access_token", session.accessToken)
	session.callDelayTimer = time.After(session.CallDelay)
	return session.PlainCall(method, params, response)
}

func (session *ServerSession) SecureCall(method string, params url.Values, response interface{}) error {
	params.Add("client_secret", session.appSecret)
	return session.AuthCall(method, params, response)
}

package hue

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Hue struct {
	IpAddress  string
	UserName   string
	DeviceType string
}

type HueError struct {
	Type        int
	Address     string
	Description string
}

func (err *HueError) Error() string {
	return fmt.Sprintf("Hue Error %v: %v %v", err.Type, err.Address, err.Description)
}

type HueAggregateError []struct {
	Error HueError
}

func (errs *HueAggregateError) Error() string {
	desc := ""
	for _, err := range *errs {
		desc = fmt.Sprintf("%v%v\n", desc, err.Error.Error())
	}
	return desc
}

// Flags for easy standard instance construction.
var ip string
var userName string
var deviceType string

func Flags() {
	flag.StringVar(&ip, "hue_ip", "192.168.1.3", "IP Address of Philips Hue hub.")
	flag.StringVar(&userName, "hue_username", "HueGoRaspberryPiUser", "Username for Hue hub.")
	flag.StringVar(&deviceType, "hue_device_type", "HueGoRaspberryPi", "Device type for Hue hub.")
}

func FromFlags() *Hue {
	return &Hue{ip, userName, deviceType}
}

func processJsonResponse(resp *http.Response, jsonBody interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err := fmt.Errorf("Http request failed: Status %d", resp.StatusCode)
		log.Printf("%v", err)
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return err
	}

	log.Printf("Got response: %v", string(body))

	// Check whether it's actually an error.
	var hueErr HueAggregateError
	if err = json.Unmarshal(body, &hueErr); err == nil {
		if len(hueErr) > 0 {
			if hueErr[0].Error.Type != 0 {
				if len(hueErr) == 1 {
					log.Printf("Request failed: %v", hueErr[0].Error.Error())
					return &(hueErr[0].Error)
				}
				log.Printf("Request failed: %v", hueErr.Error())
				return &hueErr
			}
		}
	}

	if err = json.Unmarshal(body, &jsonBody); err != nil {
		log.Printf("Failed to parse response body: %v\nerror: %v", string(body), err)
		return err
	}

	return nil
}

func (hue *Hue) get(path string, jsonBody interface{}) error {
	url := "http://" + hue.IpAddress + path

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Http GET failed: %v", err)
		return err
	}

	if err = processJsonResponse(resp, jsonBody); err != nil {
		return err
	}

	return nil
}

func (hue *Hue) post(path string, reqBody interface{}, respBody interface{}) error {
	url := "http://" + hue.IpAddress + path

	data, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Unable to create JSON for request: %v", err)
		return err
	}
	reqReader := bytes.NewReader(data)

	resp, err := http.Post(url, "application/json", reqReader)
	if err != nil {
		log.Printf("Http POST failed: %v", err)
		return err
	}

	if err = processJsonResponse(resp, respBody); err != nil {
		return err
	}

	return nil
}

func (hue *Hue) put(path string, reqBody interface{}, respBody interface{}) error {
	url := "http://" + hue.IpAddress + path

	data, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Unable to create JSON for request: %v", err)
		return err
	}
	log.Printf("Request body: %v", string(data))
	reqReader := bytes.NewReader(data)

	req, err := http.NewRequest("PUT", url, reqReader)
	if err != nil {
		log.Printf("Creating PUT request failed: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Http PUT failed: %v", err)
		return err
	}

	if err = processJsonResponse(resp, respBody); err != nil {
		return err
	}

	return nil
}

type LightState struct {
	On        bool
	Hue       int
	Sat       int
	Bri       int
	Alert     string
	ColorMode string
	CT        int
	Effect    string
	Reachable bool
	XY        []float64
}

type Light struct {
	State       LightState
	Type        string
	Name        string
	ModelId     string
	SWVersion   string
	PointSymbol map[string]string
}

// User Info

type GetUserResponse struct {
	Lights map[string]Light
	Groups map[string]interface{}
	Config struct {
		Gateway   string
		LocalTime string
		ProxyPort int
		Whitelist map[string]struct {
			LastUseDate string `json:"last use date"`
			CreateDate  string `json:"create date"`
			Name        string
		}
		SWUpdate struct {
			Notify      bool
			UpdateState int
			Url         string
			Text        string
		}
		LinkButton       bool
		PortalServices   bool
		PortalConnection string
		ProxyAddress     string
		UTC              string
		SWVersion        string
		ApiVersion       string
		Netmask          string
		Timezone         string
		PortalState      struct {
			Incoming   bool
			Outgoing   bool
			SignedOn   bool
			Connection string
		}
		Name string
		Mac  string
	}
	Schedules map[string]interface{}
	Scenes    map[string]struct {
		Name   string
		Active bool
		Lights []string
	}
}

func (hue *Hue) GetUser(resp *GetUserResponse) error {
	log.Printf("Fetching user info...")

	path := "/api/" + hue.UserName

	if err := hue.get(path, resp); err != nil {
		log.Printf("Failed to fetch user info: %v", err)
		return err
	}

	log.Printf("Got user info: %v", resp)

	return nil
}

// User Registration

type postUserRequest struct {
	Username   string `json:"username"`
	DeviceType string `json:"devicetype"`
}

type postUserResponse []struct {
	Success struct {
		Username string
	}
}

func (hue *Hue) PostUser() error {
	log.Printf("Registering user...")

	path := "/api"

	reqBody := &postUserRequest{
		hue.UserName,
		hue.DeviceType,
	}

	var respBody postUserResponse
	if err := hue.post(path, &reqBody, &respBody); err != nil {
		log.Printf("Failed to register user: %v", err)
		return err
	}

	log.Printf("Registered user: %v", respBody)

	return nil
}

// Fetching Light State

type GetLightsResponse map[string]struct {
	Name string
}

func (hue *Hue) GetLights(resp *GetLightsResponse) error {
	log.Printf("Fetching current lights...")

	path := "/api/" + hue.UserName + "/lights"

	if err := hue.get(path, resp); err != nil {
		log.Printf("Failed to fetch lights: %v", err)
		return err
	}

	log.Printf("Got lights info: %v", *resp)

	return nil
}

type GetLightResponse Light

func (hue *Hue) GetLight(id string, resp *GetLightResponse) error {
	log.Printf("Fetching light state...")

	path := "/api/" + hue.UserName + "/lights/" + id
	log.Printf("path: %v", path)

	if err := hue.get(path, resp); err != nil {
		log.Printf("Failed to fetch light: %v", err)
		return err
	}

	log.Printf("Got light info: %v", *resp)

	return nil
}

// Setting Light State

type PutLightRequest struct {
	On  *bool `json:"on,omitempty"`
	Hue *int  `json:"hue,omitempty"`
	Sat *int  `json:"sat,omitempty"`
	Bri *int  `json:"bri,omitempty"`
}

type putLightResponse []struct {
	Success map[string]interface{}
}

func (hue *Hue) PutLight(id string, state *PutLightRequest) error {
	log.Printf("Changing light state...")

	path := "/api/" + hue.UserName + "/lights/" + id + "/state"
	log.Printf("path: %v", path)

	var respBody putLightResponse
	if err := hue.put(path, state, &respBody); err != nil {
		log.Printf("Failed to change light state: %v", err)
		return err
	}

	log.Printf("Light change response: %v", respBody)

	return nil
}

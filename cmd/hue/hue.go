package main

import (
	"flag"
	. "github.com/bklimt/hue"
	"log"
	"strings"
)

func main() {
	ip := flag.String("ip", "192.168.1.3", "IP Address of Philips Hue hub.")
	userName := flag.String("username", "HueGoRaspberryPiUser", "Username for Hue hub.")
	deviceType := flag.String("device_type", "HueGoRaspberryPi", "Device type for Hue hub.")

	register := flag.Bool("register", false, "Whether to register the user with the Hue hub.")

	dumpUserInfo := flag.Bool("dump_user_info", false, "Whether to dump info about the registered user from the Hue hub.")

	on := flag.Bool("on", true, "Whether to turn the light on or off.")
	hue := flag.Int("hue", -1, "Hue to set lights to.")
	sat := flag.Int("sat", -1, "Saturation to set lights to.")
	bri := flag.Int("bri", -1, "Brightness to set lights to.")

	light := flag.String("light", "", "Light to set properties of.")

	flag.Parse()

	philipsHue := &Hue{*ip, *userName, *deviceType}

	if *register {
		if err := philipsHue.RegisterUser(); err != nil {
			if hueErr, ok := err.(*HueError); ok {
				if hueErr.Type == 101 {
					log.Fatalf("Please press the link button on the router and then try again.")
				}
			}
			log.Fatalf("Unable to register user: %v", err)
		}
	}

	if *dumpUserInfo {
		var userInfo UserInfoResponseBody
		if err := philipsHue.FetchUserInfo(&userInfo); err != nil {
			log.Fatalf("Unable to fetch user info: %v", err)
		}
	}

	lights := make([]string, 0)

	if *light != "" {
		lights = append(lights, *light)
	} else {
		var lightsInfo LightsResponseBody
		if err := philipsHue.FetchLights(&lightsInfo); err != nil {
			log.Fatalf("Unable to fetch lights: %v", err)
		}
		for lightName, _ := range lightsInfo {
			lights = append(lights, lightName)
		}
	}
	log.Printf("Controlling lights: %v", strings.Join(lights, ", "))

	state := &LightRequestBody{}
	state.On = on
	if *hue >= 0 {
		state.Hue = hue
	}
	if *sat >= 0 {
		state.Sat = sat
	}
	if *bri >= 0 {
		state.Bri = bri
	}

	for _, lightName := range lights {
		if err := philipsHue.ChangeLight(lightName, state); err != nil {
			log.Fatalf("Unable to change light %v: %v", lightName, err)
		}
	}
}

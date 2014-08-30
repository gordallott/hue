package main

import (
	"flag"
	. "github.com/bklimt/hue"
	"log"
	"strings"
)

func main() {
	Flags()

	register := flag.Bool("register", false, "Whether to register the user with the Hue hub.")

	dumpUserInfo := flag.Bool("dump_user_info", false, "Whether to dump info about the registered user from the Hue hub.")

	on := flag.Bool("on", true, "Whether to turn the light on or off.")
	hue := flag.Int("hue", -1, "Hue to set lights to.")
	sat := flag.Int("sat", -1, "Saturation to set lights to.")
	bri := flag.Int("bri", -1, "Brightness to set lights to.")

	light := flag.String("light", "", "Light to set properties of.")

	flag.Parse()

	philipsHue := FromFlags()

	if *register {
		if err := philipsHue.PostUser(); err != nil {
			if hueErr, ok := err.(*HueError); ok {
				if hueErr.Type == 101 {
					log.Fatalf("Please press the link button on the router and then try again.")
				}
			}
			log.Fatalf("Unable to register user: %v", err)
		}
	}

	if *dumpUserInfo {
		userInfo := &GetUserResponse{}
		if err := philipsHue.GetUser(userInfo); err != nil {
			log.Fatalf("Unable to fetch user info: %v", err)
		}
	}

	lights := make([]string, 0)

	if *light != "" {
		lights = append(lights, *light)
	} else {
		lightsInfo := &GetLightsResponse{}
		if err := philipsHue.GetLights(lightsInfo); err != nil {
			log.Fatalf("Unable to fetch lights: %v", err)
		}
		for lightName, _ := range *lightsInfo {
			lights = append(lights, lightName)

			lightInfo := &GetLightResponse{}
			if err := philipsHue.GetLight(lightName, lightInfo); err != nil {
				log.Fatalf("Unable to fetch light: %v", err)
			}
		}
	}
	log.Printf("Controlling lights: %v", strings.Join(lights, ", "))

	state := &PutLightRequest{}
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
		if err := philipsHue.PutLight(lightName, state); err != nil {
			log.Fatalf("Unable to change light %v: %v", lightName, err)
		}
	}
}

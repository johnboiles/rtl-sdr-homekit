package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"sort"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
)

func ftoc(f float64) float64 {
	return (f - 32) / float64(1.8)
}

// rtl_433 -F json -R 20 | go run main.go
type AmbientWeatherMessage struct {
	Battery      string  `json:"battery"`
	Channel      int     `json:"channel"`
	Device       int     `json:"device"`
	TemperatureF float64 `json:"temperature_F"`
	Model        string  `json:"model"`
	Humidity     float64 `json:"humidity"`
	TimeStr      string  `json:"time"`
}

func newAmbientWeatherSensor(msg *AmbientWeatherMessage) *ThermoHygrometer {
	info := accessory.Info{
		Name:         "Thermo Hygrometer",
		SerialNumber: fmt.Sprintf("d:%d c:%d", msg.Device, msg.Channel),
		Manufacturer: "Ambient Weather",
		Model:        msg.Model,
	}
	thermometer := NewThermoHygrometer(info, ftoc(msg.TemperatureF), -40, 60, 0.1, msg.Humidity, 10, 99, 1)
	return thermometer
}

func awmsgReader(reader *bufio.Reader) chan *AmbientWeatherMessage {
	c := make(chan *AmbientWeatherMessage)
	go func() {
		for {
			in, err := reader.ReadBytes('\n')
			if err != nil {
				log.Println("reader.ReadBytes error", err)
			}
			awmsg := AmbientWeatherMessage{}
			err = json.Unmarshal(in, &awmsg)
			if err != nil {
				log.Println("json.Unmarshal error", err)
			}
			c <- &awmsg
		}
	}()
	return c
}

func detectSensors(c chan *AmbientWeatherMessage) *map[string]*ThermoHygrometer {
	fmt.Println("Detecting sensors")
	thermometerMap := map[string]*ThermoHygrometer{}
	done := make(chan bool)
	go func() {
		time.Sleep(60 * time.Second)
		done <- true
	}()
	for {
		select {
		case awmsg := <-c:
			fmt.Printf("Detected sensor %d: %.2f°F %.1f%%rh\n", awmsg.Channel, awmsg.TemperatureF, awmsg.Humidity)
			thermometer := newAmbientWeatherSensor(awmsg)
			key := fmt.Sprintf("%d", awmsg.Channel)
			thermometerMap[key] = thermometer
		case <-done:
			done = nil
			break
		}
		if done == nil {
			break
		}
	}
	fmt.Println("Done detecting sensors")
	return &thermometerMap
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	c := awmsgReader(reader)

	thermometerMap := detectSensors(c)

	go func() {
		for {
			select {
			case awmsg := <-c:
				key := fmt.Sprintf("%d", awmsg.Channel)
				if thermoHygrometer, ok := (*thermometerMap)[key]; ok {
					fmt.Printf("Got data from sensor %d: %.2f°F %.1f%%rh\n", awmsg.Channel, awmsg.TemperatureF, awmsg.Humidity)
					temp := ftoc(float64(awmsg.TemperatureF))
					thermoHygrometer.TempSensor.CurrentTemperature.SetValue(temp)
					thermoHygrometer.HumiditySensor.CurrentRelativeHumidity.SetValue(awmsg.Humidity)
				} else {
					fmt.Println("Message from unknown sensor", key)
				}
			}
		}
	}()

	if len(*thermometerMap) == 0 {
		log.Fatal("No sensors detected")
		os.Exit(1)
	}

	keys := []string{}
	for key, _ := range *thermometerMap {
		keys = append(keys, key)
	}

	// Sort keys to maintain a consistent order for the accessories (otherwise HomeKit will switch around the sensors)
	sort.Strings(keys)

	thermometers := []*accessory.Accessory{}
	for _, key := range keys {
		thermometers = append(thermometers, (*thermometerMap)[key].Accessory)
	}

	primary := thermometers[0]
	secondary := []*accessory.Accessory{}
	if len(thermometers) > 1 {
		secondary = thermometers[1:]
	}

	fmt.Println("Primary", string(primary.Info.SerialNumber.String.Value.(string)))
	for _, t := range secondary {
		fmt.Println("Secondary", string(t.Info.SerialNumber.String.Value.(string)))
	}

	cfg := hc.Config{
		StoragePath: "cfg",
		Pin:         "32191123",
	}
	t, err := hc.NewIPTransport(cfg, primary, secondary...)
	if err != nil {
		log.Fatal(err)
	}

	hc.OnTermination(func() {
		t.Stop()
	})

	fmt.Println("Running...")
	t.Start()
	fmt.Println("Exiting...")
}

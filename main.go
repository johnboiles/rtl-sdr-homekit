package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"time"

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

func newAmbientWeatherSensor(msg *AmbientWeatherMessage) *accessory.Thermometer {
	info := accessory.Info{
		Name: "Temperature Sensor",
		//SerialNumber: fmt.Sprintf("%d-%d", msg.Device, msg.Channel),
		Manufacturer: "Ambient Weather",
		Model:        "v0",
	}
	thermometer := accessory.NewTemperatureSensor(info, msg.TemperatureF, -40, 60, 1)
	return thermometer
}

func awmsgReader(reader *bufio.Reader) chan *AmbientWeatherMessage {
	c := make(chan *AmbientWeatherMessage)
	go func() {
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
	}()
	return c
}

func detectSensors(c chan *AmbientWeatherMessage) *map[string]*accessory.Thermometer {
	fmt.Println("Detecting sensors")
	thermometerMap := map[string]*accessory.Thermometer{}
	done := make(chan bool)
	go func() {
		time.Sleep(60 * time.Second)
		done <- true
	}()
	for {
		select {
		case awmsg := <-c:
			fmt.Printf("Detected sensor %d %d: %.2f°F\n", awmsg.Device, awmsg.Channel, awmsg.TemperatureF)
			thermometer := newAmbientWeatherSensor(awmsg)
			key := fmt.Sprintf("%d-%d", awmsg.Device, awmsg.Channel)
			thermometerMap[key] = thermometer
			done = nil
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

	//thermometerMap := detectSensors(c)

	msg := AmbientWeatherMessage{
		Device:  75,
		Channel: 2,
	}
	info := accessory.Info{
		Name: "Temperature Sensor",
		//SerialNumber: fmt.Sprintf("%d-%d", msg.Device, msg.Channel),
		Manufacturer: "Ambient Weather",
		Model:        "v0",
	}
	thermometer := accessory.NewTemperatureSensor(info, msg.TemperatureF, -40, 60, 1)

	//go func() {
	//	for {
	//		select {
	//		case awmsg := <-c:
	//			key := fmt.Sprintf("%d-%d", awmsg.Device, awmsg.Channel)
	//			if thermometer, ok := (*thermometerMap)[key]; ok {
	//				fmt.Printf("Got Temp from sensor %d %d: %.2f°F", awmsg.Device, awmsg.Channel, awmsg.TemperatureF)
	//				temp := ftoc(float64(awmsg.TemperatureF))
	//				thermometer.TempSensor.CurrentTemperature.SetValue(temp)
	//			} else {
	//				fmt.Println("Message from unknown sensor", key)
	//			}
	//		}
	//	}
	//}()

	go func() {
		for {
			select {
			case awmsg := <-c:
				fmt.Printf("Got Temp from sensor %d %d: %.2f°F", awmsg.Device, awmsg.Channel, awmsg.TemperatureF)
				temp := ftoc(float64(awmsg.TemperatureF))
				thermometer.TempSensor.CurrentTemperature.SetValue(temp)
			}
		}
	}()

	//if len(*thermometerMap) == 0 {
	//	log.Fatal("No sensors detected")
	//	os.Exit(1)
	//}
	//
	//thermometers := []*accessory.Accessory{}
	//for _, value := range *thermometerMap {
	//	thermometers = append(thermometers, value.Accessory)
	//}
	//
	//primary := thermometers[0]
	//secondary := []*accessory.Accessory{}
	//if len(thermometers) > 1 {
	//	secondary = thermometers[1:]
	//}

	cfg := hc.Config{
		StoragePath: "cfg",
		Pin:         "32191123",
	}
	t, err := hc.NewIPTransport(cfg, thermometer.Accessory)
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
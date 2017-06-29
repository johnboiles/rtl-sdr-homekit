package main

import (
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"
)

type ThermoHygrometer struct {
	*accessory.Accessory

	TempSensor     *service.TemperatureSensor
	HumiditySensor *service.HumiditySensor
}

// NewThermoHygrometer returns a Thermometer & Humidity Sensor combination
func NewThermoHygrometer(info accessory.Info, temp, min, max, steps, humidity, hmin, hmax, hstep float64) *ThermoHygrometer {
	acc := ThermoHygrometer{}
	acc.Accessory = accessory.New(info, accessory.TypeThermostat)
	acc.TempSensor = service.NewTemperatureSensor()
	acc.TempSensor.CurrentTemperature.SetValue(temp)
	acc.TempSensor.CurrentTemperature.SetMinValue(min)
	acc.TempSensor.CurrentTemperature.SetMaxValue(max)
	acc.TempSensor.CurrentTemperature.SetStepValue(steps)

	acc.HumiditySensor = service.NewHumiditySensor()
	acc.HumiditySensor.CurrentRelativeHumidity.SetValue(humidity)
	acc.HumiditySensor.CurrentRelativeHumidity.SetMinValue(hmin)
	acc.HumiditySensor.CurrentRelativeHumidity.SetMaxValue(hmax)
	acc.HumiditySensor.CurrentRelativeHumidity.SetStepValue(hstep)

	acc.AddService(acc.TempSensor.Service)
	acc.AddService(acc.HumiditySensor.Service)

	return &acc
}

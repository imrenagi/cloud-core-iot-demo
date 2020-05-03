package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"google.golang.org/api/cloudiot/v1"
)

var memory map[string]*Device = make(map[string]*Device, 0)

type SensorData struct {
	Temperature int `json:"temperature"`
	Humidity    int `json:"humidity"`
}

var maxCap = 10

func NewDevice(id, registryID, location, projectID string) *Device {
	return &Device{
		ID:          id,
		ProjectID:   projectID,
		RegistryID:  registryID,
		Location:    location,
		temperature: make([]int, 0),
		Config: Configuration{
			MinTemperature: 20,
			MaxTemperature: 30,
			PingPeriodSec:  10,
		},
	}
}

// Device ...
type Device struct {
	ID         string
	ProjectID  string
	RegistryID string
	Location   string

	temperature []int
	IsAirConON  bool
	Config      Configuration
}

func (d Device) String() string {
	return fmt.Sprintf("device %s - avg %f - ac %v", d.ID, d.AvgTemperature(), d.IsAirConON)
}

// RecordTemperature ...
func (d *Device) RecordTemperature(val int) error {
	l := len(d.temperature)
	if l >= maxCap {
		d.temperature = d.temperature[1:]
	}
	d.temperature = append(d.temperature, val)
	return nil
}

type commandSender interface {
	SendCommandToDevice(name string, sendcommandtodevicerequest *cloudiot.SendCommandToDeviceRequest) *cloudiot.ProjectsLocationsRegistriesDevicesSendCommandToDeviceCall
}

// TurnAirCon ...
func (d *Device) TurnAirCon(ctx context.Context, on bool, sender commandSender) error {

	cmd := struct {
		IsAcOn bool `json:"is_ac_on"`
	}{
		IsAcOn: on,
	}

	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	req := cloudiot.SendCommandToDeviceRequest{
		BinaryData: base64.StdEncoding.EncodeToString(b),
	}

	_, err = sender.SendCommandToDevice(d.path(), &req).Do()
	if err != nil {
		return err
	}

	fmt.Println("Sent command to device")
	return nil
}

func (d Device) path() string {
	return fmt.Sprintf("projects/%s/locations/%s/registries/%s/devices/%s", d.ProjectID, d.Location, d.RegistryID, d.ID)
}

type configUpdater interface {
	ModifyCloudToDeviceConfig(name string, modifycloudtodeviceconfigrequest *cloudiot.ModifyCloudToDeviceConfigRequest) *cloudiot.ProjectsLocationsRegistriesDevicesModifyCloudToDeviceConfigCall
}

func (d *Device) Configure(newConf Configuration, updater configUpdater) error {

	b, err := json.Marshal(newConf)
	if err != nil {
		return err
	}

	req := cloudiot.ModifyCloudToDeviceConfigRequest{
		BinaryData: base64.StdEncoding.EncodeToString(b),
	}

	response, err := updater.ModifyCloudToDeviceConfig(d.path(), &req).Do()
	if err != nil {
		return err
	}

	d.Config = newConf

	fmt.Printf("Config set!\nVersion now: %d\n", response.Version)
	return nil
}

func (d *Device) AvgTemperature() float64 {
	sum := 0
	for _, v := range d.temperature {
		sum += v
	}
	return float64(sum) / float64(len(d.temperature))
}

type Configuration struct {
	MinTemperature int `json:"min_temp"`
	MaxTemperature int `json:"max_temp"`
	PingPeriodSec  int `json:"ping_period"`
}

func (c Configuration) String() string {
	return fmt.Sprintf("min: %d, max: %d, ping: %d", c.MinTemperature, c.MaxTemperature, c.PingPeriodSec)
}

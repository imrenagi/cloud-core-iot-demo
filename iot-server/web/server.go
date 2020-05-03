package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudiot/v1"
)

//NewServer creates new server used for accessing the audit log
func NewServer() *Server {

	projectID := "imre-demo"

	ctx := context.Background()
	pubsub, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		panic(err)
	}

	httpClient, err := google.DefaultClient(ctx, cloudiot.CloudPlatformScope)
	if err != nil {
		panic(err)
	}
	iot, err := cloudiot.New(httpClient)
	if err != nil {
		panic(err)
	}

	s := &Server{
		Router:         mux.NewRouter(),
		pubsubClient:   pubsub,
		cloudiotClient: iot,
	}

	go s.subscribeSensorTelemetry(context.Background())
	go s.subscribeSensorState(context.Background())

	s.routes()
	return s
}

// GetHandler returns http.Handler which intercepted by the cors checker.
func (s *Server) GetHandler() http.Handler {
	return s.Router
}

//Server is just a server
type Server struct {
	Router         *mux.Router
	pubsubClient   *pubsub.Client
	cloudiotClient *cloudiot.Service
}

func (s Server) routes() {
	s.Router.HandleFunc("/devices/{device_id}/config", s.ChangeDeviceConfig()).Methods("POST")
	s.Router.HandleFunc("/devices/{device_id}/command", s.SendCommand()).Methods("POST")
}

// ChangeDeviceState ...
func (s *Server) ChangeDeviceConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		deviceID := vars["device_id"]

		var mu sync.Mutex
		mu.Lock()
		defer mu.Unlock()

		var cfg Configuration
		err := json.NewDecoder(r.Body).Decode(&cfg)
		if err != nil {
			fmt.Println(err)
			return
		}

		if d, ok := memory[deviceID]; ok {
			err := d.Configure(cfg, s.cloudiotClient.Projects.Locations.Registries.Devices)
			if err != nil {
				fmt.Println(err)
				return
				// return err
			}

			// if this is happened, then we should fetch the device information to google cloud
		}
		writeSuccessResponse(w, http.StatusOK, empty{})
	}
}

func (s *Server) SendCommand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		deviceID := vars["device_id"]

		var mu sync.Mutex
		mu.Lock()
		defer mu.Unlock()

		type state struct {
			IsAcOn bool `json:"ac"`
		}
		var st state
		err := json.NewDecoder(r.Body).Decode(&st)
		if err != nil {
			fmt.Println(err)
			return
		}

		if d, ok := memory[deviceID]; ok {
			err := d.TurnAirCon(r.Context(), st.IsAcOn, s.cloudiotClient.Projects.Locations.Registries.Devices)
			if err != nil {
				fmt.Println(err)
				return
				// return err
			}

			// if this is happened, then we should fetch the device information to google cloud
		}
		writeSuccessResponse(w, http.StatusOK, empty{})
	}
}

type empty struct {
}

func (s *Server) subscribeSensorTelemetry(ctx context.Context) error {
	var mu sync.Mutex
	subID := "testing"
	sub := s.pubsubClient.Subscription(subID)
	ctx, cancel := context.WithCancel(ctx)
	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {

		var sensorReading SensorData
		err := json.Unmarshal(msg.Data, &sensorReading)
		if err != nil {
			cancel()
		}

		mu.Lock()
		defer mu.Unlock()

		deviceID := msg.Attributes["deviceId"]
		registryID := msg.Attributes["deviceRegistryId"]
		location := msg.Attributes["deviceRegistryLocation"]
		projectID := msg.Attributes["projectId"]
		var device *Device
		if d, ok := memory[deviceID]; ok {
			device = d
		} else {
			device = NewDevice(deviceID, registryID, location, projectID)
		}

		err = device.RecordTemperature(sensorReading.Temperature)
		if err != nil {
			cancel()
		}

		memory[deviceID] = device
		fmt.Println(device.String())

		if device.AvgTemperature() >= 25 && !device.IsAirConON {
			err := device.TurnAirCon(ctx, true, s.cloudiotClient.Projects.Locations.Registries.Devices)
			if err != nil {
				fmt.Println(err)
				// cancel()
			}
		} else if device.AvgTemperature() < 25 && device.IsAirConON {
			err := device.TurnAirCon(ctx, false, s.cloudiotClient.Projects.Locations.Registries.Devices)
			if err != nil {
				fmt.Println(err)
				// cancel()
			}
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("Receive: %v", err)
	}
	return nil
}

func (s *Server) subscribeSensorState(ctx context.Context) error {

	subID := "state-subscription"
	sub := s.pubsubClient.Subscription(subID)
	ctx, cancel := context.WithCancel(ctx)
	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var mu sync.Mutex

		type state struct {
			IsAcOn bool `json:"ac"`
		}
		var st state
		err := json.Unmarshal(msg.Data, &st)
		if err != nil {
			cancel()
		}

		mu.Lock()
		defer mu.Unlock()

		deviceID := msg.Attributes["deviceId"]
		if d, ok := memory[deviceID]; ok {
			d.IsAirConON = st.IsAcOn
			memory[deviceID] = d
			fmt.Fprintf(os.Stdout, "Device State Updated: %s\n", d.String())
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("Receive: %v", err)
	}
	return nil
}

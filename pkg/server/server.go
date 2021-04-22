package server

import (
	"time"
	"log"
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/massn/pms5003onGo/pkg/util"
	"github.com/massn/pms5003onGo/pkg/device"
)

const timeBuffer = 10

var timeout int
var devicePortName string
var semaphore chan struct{} = make(chan struct{}, 1)
var savedOutput string = "No data yet"

type serverData struct {
	PMS5003 *device.Data `json:"pms5003"`
}

func Start(period, timeoutArg int, port, devicePortNameArg string)error{
	timeout = timeoutArg
	devicePortName = devicePortNameArg

	if period + timeBuffer < timeout {
		msg := "too short period compared to timeout"
		log.Println(msg)
		return fmt.Errorf(msg)
	}

	go getDataPeriodically(period)

	http.HandleFunc("/", handler)
	log.Printf("starting server at %s\n", port)
	http.ListenAndServe(":"+port,nil)
	return nil
}

func getDataPeriodically(period int){
	ticker := time.NewTicker(time.Duration(period) * time.Second)
	for {
		fmt.Println("server waiting ticker message.")
		select{
		case <-ticker.C:
			log.Println("getting data by ticker")
			data := util.GetDataInTime(timeout, devicePortName)
			if data.Err != nil {
				log.Println("not update the savedOutput")
				break
			}
			s := &serverData{PMS5003: data}
			marshaled, err := json.MarshalIndent(s, "", "    ")
			if err != nil {
				log.Printf("failed to marshal to json. reason:%v\n",err)
			    break
			}
			savedOutput = string(marshaled)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("get request:%v\n", r)
	fmt.Fprintf(w, savedOutput)
}

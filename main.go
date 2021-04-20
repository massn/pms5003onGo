package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"github.com/massn/pms5003onGo/pkg/device"
	"github.com/massn/pms5003onGo/pkg/server"
	"github.com/massn/pms5003onGo/pkg/util"
)

const (
	defaultTimeoutSeconds = 5
	defaultDevicePortName       = "/dev/ttyAMA0"
	defaultPort = "8080"
)

func main() {
	var (
		v = flag.Bool("v", false, "verbose output.")
		t = flag.Int("t", defaultTimeoutSeconds, "timeout seconds.")
		d = flag.String("d", defaultDevicePortName, "device port to read.")
		j = flag.Bool("j", false, "json output.")
		s = flag.Bool("s", false, "server mode.")
		p = flag.String("p", defaultPort, "server port.")
	)
	flag.Parse()
	if !(*v) {
		log.SetOutput(ioutil.Discard)
	}

	if *s {
		server.Start(60, *t, *p,*d)
	}else{
		commandMode(*t, *d, *j)
	}
}

func commandMode(timeout int, devicePortName string, jsonOutput bool){
	data := util.GetDataInTime(timeout, devicePortName)
	if jsonOutput {
		s, err := json.MarshalIndent(data, "", "    ")
		if err != nil{
			log.Fatalf("failed to marshal to json. reason:%v\n",err)
		}
		fmt.Println(string(s))
	}else{
		if data.Err != nil {
			fmt.Printf("failed to get data. reason:%v\n",data.Err)
		}else{
			printResults(data)
		}
	}
}

func printResults(d *device.Data) {
	micron := "um"
	unitM3 := "ug/m^3"
	unitL := "1/0.1L"
	data := [][]string{
		{"PM1.0", strconv.Itoa(d.PM1p0), unitM3},
		{"PM2.5", strconv.Itoa(d.PM2p5), unitM3},
		{"PM10", strconv.Itoa(d.PM10), unitM3},
		{"PM1.0 in atmos env", strconv.Itoa(d.PM1p0_atmos), unitM3},
		{"PM2.5 in atmos env", strconv.Itoa(d.PM2p5_atmos), unitM3},
		{"PM10 in atmos env", strconv.Itoa(d.PM10_atmos), unitM3},
		{"0.3" + micron, strconv.Itoa(d.D0p3), unitL},
		{"0.5" + micron, strconv.Itoa(d.D0p5), unitL},
		{"1.0" + micron, strconv.Itoa(d.D1p0), unitL},
		{"2.5" + micron, strconv.Itoa(d.D2p5), unitL},
		{"5.0" + micron, strconv.Itoa(d.D5p0), unitL},
		{"10" + micron, strconv.Itoa(d.D10p0), unitL},
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Data", "Number", "Unit"})
	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}

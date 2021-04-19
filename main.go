package main

import (
	"sync"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
	"github.com/massn/pms5003onGo/pkg/device"
)

const (
	defaultTimeoutSeconds = 5
	defaultPortPath       = "/dev/ttyAMA0"
)

func main() {
	var (
		v = flag.Bool("v", false, "verbose output.")
		t = flag.Int("t", defaultTimeoutSeconds, "timeout seconds.")
		p = flag.String("p", defaultPortPath, "port to read.")
		j = flag.Bool("j", false, "json output.")
	)
	flag.Parse()
	if !(*v) {
		log.SetOutput(ioutil.Discard)
	}

	dataChan := make(chan *device.Data)
	defer close(dataChan)
	quitChan := make(chan struct{})
	defer close(quitChan)
	var wg sync.WaitGroup

	wg.Add(1)
	d, err := device.New(*p, &wg)
	if err != nil{
		panic(err)
	}
	go device.GetData(d, dataChan, quitChan)
	select {
	case data := <-dataChan:
		if *j {
			s, err := json.MarshalIndent(data, "", "    ")
			if err != nil{
				log.Fatalf("failed to marshal to json. reason:%v\n",err)
			}
			fmt.Println(string(s))
		}else{
			if data.Err != nil {
				fmt.Printf("failed to get data. reason : %v\n",data.Err)
			}else{
				printResults(data)
			}
	}
	case <-time.After((time.Duration)(*t) * time.Second):
		fmt.Printf("%d seconds elapsed. timeout.\n", *t)
		quitChan <- struct{}{}
	}
	wg.Wait()
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

package util

import(
	"fmt"
	"time"
	"log"
	"sync"
	"github.com/massn/pms5003onGo/pkg/device"
)

func GetDataInTime(timeout int, portPath string)*device.Data{
	dataChan := make(chan *device.Data)
	defer close(dataChan)
	quitChan := make(chan struct{})
	defer close(quitChan)
	var wg sync.WaitGroup
	resultData := &device.Data{}

	d, err := device.New(portPath, &wg)
	if err != nil{
		panic(err)
	}
	go device.GetData(d, dataChan, quitChan)
	select {
	case resultData = <-dataChan:
	case <-time.After((time.Duration)(timeout) * time.Second):
		msg := fmt.Sprintf("%d seconds elapsed. timeout.", timeout)
		log.Println(msg)
		quitChan <- struct{}{}
		resultData.Err = fmt.Errorf(msg)
	}
	wg.Wait()
	return resultData
}

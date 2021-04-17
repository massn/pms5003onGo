package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"github.com/olekukonko/tablewriter"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

const defaultTimeoutSeconds = 5

type readingData struct {
	acc     int
	port    io.ReadWriteCloser
	started bool
}

type twoBytesData struct {
	num  int
	high byte
	low  byte
}

func main() {
	var (
		v = flag.Bool("v", false, "verbose output")
		t = flag.Int("t", defaultTimeoutSeconds, "timeout seconds")
	)
	flag.Parse()
	if !(*v) {
		log.SetOutput(ioutil.Discard)
	}
	resultsChan := make(chan []twoBytesData)
	go getData(resultsChan)
	select {
	case results := <-resultsChan:
		printResults(results)
	case <-time.After((time.Duration)(*t) * time.Second):
		fmt.Printf("%d seconds elapsed. timeout.\n", *t)
	}
}

func getData(resultsChan chan []twoBytesData) {
	options := serial.OpenOptions{
		PortName:              "/dev/ttyAMA0",
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 100,
	}

	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}
	defer port.Close()

	data := &readingData{acc: 0, port: port, started: false}

	results := make([]twoBytesData, 13, 13)
	for {
		waitForStarting(data)

		log.Println("started to read")

		if err := setFrameLength(data); err != nil {
			log.Printf("failed to set the frame length. reason:%v\n", err)
			continue
		}
		log.Println("get the frame length")

		b := readExactBytes(28, port)

		for i := 0; i <= 12; i++ {
			d := get2BytesData(b[i*2 : i*2+2])
			data.acc = data.acc + int(d.high) + int(d.low)
			log.Printf("%d data: %#v\n", i, d)
			results[i] = d
		}
		c := get2BytesData(b[26:28])
		log.Printf("acc:%d, checksum:%d\n", data.acc, c.num)
		if int(data.acc) == c.num {
			break
		}
	}
	resultsChan <- results
}

func printResults(r []twoBytesData) {
	if len(r) != 13 {
		log.Fatalf("%d bad results length\n", len(r))
	}
	data := [][]string{
		[]string{"PM1.0", strconv.Itoa(r[0].num), "ug/m^3"},
		[]string{"PM2.5", strconv.Itoa(r[1].num), "ug/m^3"},
		[]string{"PM10", strconv.Itoa(r[2].num), "ug/m3"},
		[]string{"PM1.0 in atomos env", strconv.Itoa(r[3].num), "ug/m^3"},
		[]string{"PM2.5 in atmos env", strconv.Itoa(r[4].num), "ug/m^3"},
		[]string{"PM10 in atmos env", strconv.Itoa(r[5].num), "ug/m^3"},
		[]string{"0.3um", strconv.Itoa(r[6].num), "1/0.1L"},
		[]string{"0.5um", strconv.Itoa(r[7].num), "1/0.1L"},
		[]string{"1.0um", strconv.Itoa(r[8].num), "1/0.1L"},
		[]string{"2.5um", strconv.Itoa(r[9].num), "1/0.1L"},
		[]string{"5.0um", strconv.Itoa(r[10].num), "1/0.1L"},
		[]string{"10um", strconv.Itoa(r[11].num), "1/0.1L"},
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Data", "Number", "Unit"})
	for _, v := range data {
		table.Append(v)
	}
	table.Render()
}

func waitForStarting(data *readingData) {
	for {
		b := make([]byte, 1, 1)
		n, err := data.port.Read(b)

		if err != nil {
			log.Fatalf("failed to read port. reason:%v\n", err)
		}
		if n != 1 {
			log.Fatalf("failed to read 1 byte")
		}
		if !data.started {
			if b[0] == byte(0x42) {
				log.Println("get 0x42")
				data.started = true
				data.acc += int(0x42)
			}
		} else if b[0] == byte(0x4d) {
			log.Println("get 0x4d")
			data.acc += int(0x4d)
			break
		} else {
			data.started = false
			data.acc = 0
		}
	}
}

func setFrameLength(data *readingData) error {
	frameLength, err := read2bytes(data.port)
	if err != nil {
		log.Println("failed to read the frame length.")
		return err
	}
	if frameLength.num != 28 {
		log.Printf("%d is a bad frame length\n", frameLength.num)
		return fmt.Errorf("failed to get the right frame length.")
	}
	data.acc = data.acc + int(frameLength.high) + int(frameLength.low)
	return nil
}

func read2bytes(port io.ReadWriteCloser) (twoBytesData, error) {
	b := readExactBytes(2, port)
	log.Printf("read %d(0x%x) %d(0x%x)\n", b[0], b[0], b[1], b[1])
	return get2BytesData(b), nil
}

func get2BytesData(twoBytes []byte) twoBytesData {
	var read uint16
	buf := bytes.NewReader(twoBytes)
	if err := binary.Read(buf, binary.BigEndian, &read); err != nil {
		log.Fatalf("failed ro read %x %x. reason:%v\n", twoBytes[0], twoBytes[1], err)
	}
	return twoBytesData{num: int(read), high: twoBytes[0], low: twoBytes[1]}
}

func readExactBytes(n int, port io.ReadWriteCloser) []byte {
	b := make([]byte, 1, 1)
	result := make([]byte, n, n)
	for i := 0; i < n; i++ {
		for {
			readNum, err := port.Read(b)
			if err != nil {
				log.Fatalf("failed to read. reason:%v\n", err)
			}
			log.Printf("read %x(0x%v)\n", b[0], uint16(b[0]))
			if readNum < 1 {
				log.Println("failed to read 1 byte")
				log.Println("waiting 1 second\n")
				time.Sleep(time.Second)
				continue
			}
			break
		}
		result[i] = b[0]
	}
	return result
}

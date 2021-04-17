package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"io"
	"log"
	"strconv"
	"time"
)

const waitSeconds = 2

type readingData struct {
	acc     int
	port    io.ReadWriteCloser
	started bool
}

type twoBytesData struct {
	num  uint16
	high byte
	low  byte
}

func main() {
	// Set up options.
	options := serial.OpenOptions{
		PortName:              "/dev/ttyAMA0",
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		InterCharacterTimeout: 100,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}
	// Make sure to close it later.
	defer port.Close()

	data := &readingData{acc: 0, port: port, started: false}

	for {
		waitForStarting(data)

		log.Println("started to read")

		if err := setFrameLength(data); err != nil {
			log.Printf("failed to set the frame length. reason:%v\n", err)
			continue
		}
		log.Println("get the frame length")

		b := readExactBytes(28, port)
		log.Printf("size of b : %d\n", len(b))

		for i := 0; i <= 12; i++ {
			d := get2BytesData(b[i*2 : i*2+2])
			data.acc = data.acc + int(d.high) + int(d.low)
			log.Printf("%d data: %#v\n", i, d)
		}
		c := get2BytesData(b[26:28])
		log.Printf("checksum : %#v\n", c)
		log.Printf("acc:%d, checksum:%d\n", data.acc, c.num)
		log.Printf("acc:%d, checksum:%d\n",
			strconv.FormatInt(int64(data.acc), 2),
			strconv.FormatInt(int64(c.num), 2))
		if int(data.acc) == int(c.num) {
			break
		}
	}

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
		} else if b[0] == byte(0x00) {
			continue
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
	return twoBytesData{num: uint16(read), high: twoBytes[0], low: twoBytes[1]}
}

func readExactBytes(n int, port io.ReadWriteCloser) []byte {
	for {
		b := make([]byte, n, n)
		readNum, err := port.Read(b)
		if err != nil {
			log.Fatalf("failed to read. reason:%v\n", err)
		}
		log.Printf("read %v\n", b)
		if readNum < n {
			log.Printf("failed to read %d bytes. read just %d bytes.\n", n, readNum)
			log.Printf("waiting %d seconds\n", waitSeconds)
			time.Sleep(time.Second * waitSeconds)
			return append(b[0:readNum], readExactBytes(n-readNum, port)...)
		} else {
			log.Printf("read %d bytes.\n", n)
			return b
		}
	}
}

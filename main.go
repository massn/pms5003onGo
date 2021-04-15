package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	"io"
	"log"
)

func main() {
	// Set up options.
	options := serial.OpenOptions{
		PortName:        "/dev/ttyAMA0",
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	// Make sure to close it later.
	defer port.Close()

	acc := 0
	started := false
	for {
		b := make([]byte, 1, 1)
		n, err := port.Read(b)

		if err != nil {
			log.Fatalf("port.Read: %v", err)
		}
		if n != 1 {
			log.Fatalf("Failed to read 1 byte")
		}
		if !started {
			if b[0] == byte(0x42) {
				started = true
				acc += int(0x42)
			}
		} else if b[0] == byte(0x4d) {
			acc += int(0x4d)
			break
		} else {
			started = false
			acc = 0
		}
	}
	fmt.Println("start to read")
	frameLength, err := read2bytes(port)
	if err != nil {
		log.Fatalf("Failed to read the frame length.")
	}
	if frameLength.num != 28 {
		log.Fatalf("%d is bad frame length", frameLength)
	}
	acc = acc + int(frameLength.high) + int(frameLength.low)
	for i := 1; i <= 13; i++ {
		data, err := read2bytes(port)
		if err != nil {
			log.Fatalf("Failed to read data. %v", err)
		}
		acc = acc + int(data.high) + int(data.low)
		fmt.Printf("%d:%d acc:%d\n", i, data.num, acc)
	}
	checkBits, err := read2bytes(port)
	if err != nil {
		log.Fatalf("Failed to read check bits. %v", err)
	}

	fmt.Printf("check bits:%d\n", checkBits)
	if int(checkBits.num) == acc {
		fmt.Println("Checked sum!!")
	} else {
		log.Fatalf("Failed to check sum!")
	}
}

type data struct {
	num  uint16
	high byte
	low  byte
}

func read2bytes(port io.ReadWriteCloser) (data, error) {
	b := make([]byte, 2, 2)
	_, err := port.Read(b)

	if err != nil {
		log.Fatalf("port.Read: %v", err)
		return data{}, err
	}
	//if n != 2 {
	//log.Fatalf("Failed to read 2 bytes. Read %d bytes.", n)
	//return 0, err
	//}
	fmt.Printf("read %d(0x%x) %d(0x%x)\n", b[0], b[0], b[1], b[1])
	var read uint16

	buf := bytes.NewReader(b)
	if err := binary.Read(buf, binary.BigEndian, &read); err != nil {
		log.Fatalf("binary.Read failed:%v", err)
		return data{}, err
	}
	return data{num: uint16(read), high: b[0], low: b[1]}, nil
}

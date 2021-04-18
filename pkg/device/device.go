package device

import(
	"time"
	"encoding/binary"
	"bytes"
	"fmt"
	"io"
	"log"
	"github.com/jacobsa/go-serial/serial"
)


type Data struct {
	PM1p0       int
	PM2p5       int
	PM10        int
	PM1p0_atmos int
	PM2p5_atmos int
	PM10_atmos  int
	B0p3        int
	B0p5        int
	B1p0        int
	B2p5        int
	B5p0        int
	B10p0       int
}

type state struct {
	acc     int
	port    io.ReadWriteCloser
	started bool
}

type twoBytesData struct {
	num  int
	high byte
	low  byte
}

func GetData(portPath string, dataChan chan *Data) {
	options := serial.OpenOptions{
		PortName:              portPath,
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

	rd := &state{acc: 0, port: port, started: false}

	tmp := make([]int, 13, 13)
	for {
		waitForStarting(rd)

		log.Println("started to read")

		if err := setFrameLength(rd); err != nil {
			log.Printf("failed to set the frame length. reason:%v\n", err)
			continue
		}
		log.Println("get the frame length")

		b := readExactBytes(28, port)

		for i := 0; i <= 12; i++ {
			d := get2BytesData(b[i*2 : i*2+2])
			rd.acc = rd.acc + int(d.high) + int(d.low)
			log.Printf("%d data: %#v\n", i, d)
			tmp[i] = d.num
		}
		c := get2BytesData(b[26:28])
		log.Printf("acc:%d, checksum:%d\n", rd.acc, c.num)
		if int(rd.acc) == c.num {
			break
		}
	}
	data := &Data{
		PM1p0:       tmp[0],
		PM2p5:       tmp[1],
		PM10:        tmp[2],
		PM1p0_atmos: tmp[3],
		PM2p5_atmos: tmp[4],
		PM10_atmos:  tmp[5],
		B0p3:        tmp[6],
		B0p5:        tmp[7],
		B1p0:        tmp[8],
		B2p5:        tmp[9],
		B5p0:        tmp[10],
		B10p0:       tmp[11],
	}
	dataChan <- data
}

func waitForStarting(s *state) {
	for {
		b := make([]byte, 1, 1)
		n, err := s.port.Read(b)

		if err != nil {
			log.Fatalf("failed to read port. reason:%v\n", err)
		}
		if n != 1 {
			log.Fatalf("failed to read 1 byte")
		}
		if !s.started {
			if b[0] == byte(0x42) {
				log.Println("get 0x42")
				s.started = true
				s.acc += int(0x42)
			}
		} else if b[0] == byte(0x4d) {
			log.Println("get 0x4d")
			s.acc += int(0x4d)
			break
		} else {
			s.started = false
			s.acc = 0
		}
	}
}

func setFrameLength(s *state) error {
	frameLength, err := read2bytes(s.port)
	if err != nil {
		log.Println("failed to read the frame length.")
		return err
	}
	if frameLength.num != 28 {
		log.Printf("%d is a bad frame length\n", frameLength.num)
		return fmt.Errorf("failed to get the right frame length.")
	}
	s.acc =s.acc + int(frameLength.high) + int(frameLength.low)
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
				log.Println("waiting 1 second")
				time.Sleep(time.Second)
				continue
			}
			break
		}
		result[i] = b[0]
	}
	return result
}

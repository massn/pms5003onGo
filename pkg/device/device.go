package device

import(
	"sync"
	"time"
	"encoding/binary"
	"bytes"
	"fmt"
	"io"
	"log"
	"github.com/jacobsa/go-serial/serial"
)


type Data struct {
	PM1p0       int `json:"pm1.0"`
	PM2p5      int `json:"pm2.5"`
	PM10        int `json:"pm10"`
	PM1p0_atmos int `json:"pm1.0atmos"`
	PM2p5_atmos int `json:"pm2.5atmos"`
	PM10_atmos  int `json:"pm10atmos"`
	D0p3        int `json:"dia0.3um"`
	D0p5        int `json:"dia0.5um"`
	D1p0        int `json:"dia1.0um"`
	D2p5        int `json:"dia2.5um"`
	D5p0      int `json:"dia5.0um"`
	D10p0     int `json:"dia10.0um"`
}

type state struct {
	acc     int
	port    io.ReadWriteCloser
	started bool
	wg *sync.WaitGroup
}

type twoBytesData struct {
	num  int
	high byte
	low  byte
}

func New(portPath string, wg *sync.WaitGroup)*state{
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
	return &state{acc:0, port: port, started:false, wg: wg}
}

func (s *state)Close(){
	s.port.Close()
}

func GetData(s *state, dataChan chan *Data, quit chan struct{}) {
	defer s.wg.Done()
	defer s.Close()
	tmp := make([]int, 13, 13)
	for {
		select {
		case <- quit:
			return
		default:
		}
		started := make(chan struct{})
		defer close(started)
		quitWFS := make(chan struct{})
		defer close(quitWFS)
		s.wg.Add(1)
		go waitForStarting(s, started, quitWFS)
		select {
		case <- quit:
			quitWFS <- struct{}{}
			return
		case <- started:
		}
		log.Println("started to read")
		if err := setFrameLength(s); err != nil {
			log.Printf("failed to set the frame length. reason:%v\n", err)
			continue
		}
		log.Println("get the frame length")
		b := readExactBytes(28, s.port)
		for i := 0; i <= 12; i++ {
			d := get2BytesData(b[i*2 : i*2+2])
			s.acc = s.acc + int(d.high) + int(d.low)
			log.Printf("%d data: %#v\n", i, d)
			tmp[i] = d.num
		}
		c := get2BytesData(b[26:28])
		log.Printf("acc:%d, checksum:%d\n", s.acc, c.num)
		if int(s.acc) == c.num {
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
		D0p3:        tmp[6],
		D0p5:        tmp[7],
		D1p0:        tmp[8],
		D2p5:        tmp[9],
		D5p0:        tmp[10],
		D10p0:       tmp[11],
	}
	dataChan <- data
}

func waitForStarting(s *state, started, quit chan struct{}){
	defer s.wg.Done()
	for {
		select {
		case <- quit:
			return
		default:
		}

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
			started <- struct{}{}
			return
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

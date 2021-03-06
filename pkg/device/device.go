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
	PM1p0       int `json:"pm1p0"`
	PM2p5      int `json:"pm2p5"`
	PM10        int `json:"pm10"`
	PM1p0_atmos int `json:"pm1p0atmos"`
	PM2p5_atmos int `json:"pm2p5atmos"`
	PM10_atmos  int `json:"pm10atmos"`
	D0p3        int `json:"dia0p3um"`
	D0p5        int `json:"dia0p5um"`
	D1p0        int `json:"dia1p0um"`
	D2p5        int `json:"dia2p5um"`
	D5p0      int `json:"dia5p0um"`
	D10p0     int `json:"dia10um"`
	Err error
}

type state struct {
	acc     int
	port    io.ReadWriteCloser
	wg *sync.WaitGroup
}

func (s *state)reset(){
	s.acc = 0
}

type twoBytesData struct {
	num  int
	high byte
	low  byte
}

func New(portPath string, wg *sync.WaitGroup)(*state, error){
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
		log.Printf("failed to open port. reason : %v\n", err)
		return &state{}, err
	}
	wg.Add(1)
	return &state{acc:0, port: port, wg: wg}, nil
}

func (s *state)Stop(){
	s.port.Close()
	s.wg.Done()
}

func GetData(s *state, dataChan chan *Data, quit chan struct{}) {
	defer s.Stop()
	tmp := make([]int, 13, 13)
	var tmpErr error
	L: for {
		select {
		case <- quit:
			dataChan <- &Data{Err:fmt.Errorf("timeout")}
			return
		default:
		}
		startErr := make(chan error)
		defer close(startErr)
		quitWFS := make(chan struct{})
		defer close(quitWFS)
		var wgs sync.WaitGroup
		wgs.Add(1)
		go waitForStarting(s, &wgs, startErr, quitWFS)
		select {
		case <- quit:
			log.Println("got quit from")
			quitWFS <- struct{}{}
		    wgs.Wait()
			return
		case err :=<-startErr:
			log.Printf("got startErr:%v\n", err)
			if err != nil{
				tmpErr = err
		        wgs.Wait()
				break L
			}
		}
		wgs.Wait()
		log.Println("started to read")
		if err := setFrameLength(s); err != nil {
			log.Printf("failed to set the frame length. reason:%v\n", err)
			s.reset()
			continue
		}
		log.Println("set the frame length")
		b, err := readExactBytes(28, s.port)
		if err != nil{
			tmpErr = err
			break
		}
		for i := 0; i <= 12; i++ {
			d, err := get2BytesData(b[i*2 : i*2+2])
			if err != nil{
				tmpErr = err
				break
			}
			s.acc = s.acc + int(d.high) + int(d.low)
			log.Printf("%d data: %#v\n", i, d)
			tmp[i] = d.num
		}
		c, err := get2BytesData(b[26:28])
		if err != nil{
			tmpErr = err
			break
		}
		log.Printf("acc:%d, checksum:%d\n", s.acc, c.num)
		if int(s.acc) == c.num {
			tmpErr = nil
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
		Err: tmpErr,
	}
	dataChan <- data
}

func waitForStarting(s *state, startWG *sync.WaitGroup, startErr chan error, quit chan struct{}){
	defer startWG.Done()
	got42 := false
	b := make([]byte, 1, 1)
	for {
		select {
		case <- quit:
			return
		default:
		}

		n, err := s.port.Read(b)

		if err != nil {
			startErr <- fmt.Errorf("failed to read port. reason:%v\n", err)
			return
		}
		if n != 1 {
			startErr <- fmt.Errorf("failed to read 1 byte")
			return
		}
		if !got42 {
			if b[0] == byte(0x42) {
				log.Println("get 0x42")
				got42 = true
				s.acc += int(0x42)
			}
		} else {
			if b[0] == byte(0x4d) {
				log.Println("get 0x4d")
				s.acc += int(0x4d)
				startErr <- nil
				return
			} else {
				got42 = false
				s.acc = 0
			}
		}
	}
}

func setFrameLength(s *state) error {
	frameLength, err := read2bytes(s.port)
	if err != nil {
		log.Println("failed to read the frame length.")
		s.reset()
		return err
	}
	if frameLength.num != 28 {
		log.Printf("%d is a bad frame length\n", frameLength.num)
		s.reset()
		return fmt.Errorf("failed to get the right frame length.")
	}
	s.acc =s.acc + int(frameLength.high) + int(frameLength.low)
	return nil
}

func read2bytes(port io.ReadWriteCloser) (twoBytesData, error) {
	b, err := readExactBytes(2, port)
	if err != nil {
		return twoBytesData{}, err
	}
	log.Printf("read %d(0x%x) %d(0x%x)\n", b[0], b[0], b[1], b[1])
	d, err := get2BytesData(b)
	if err != nil {
		return twoBytesData{}, err
	}
	return d, nil
}

func get2BytesData(twoBytes []byte) (twoBytesData, error) {
	var read uint16
	buf := bytes.NewReader(twoBytes)
	if err := binary.Read(buf, binary.BigEndian, &read); err != nil {
		log.Printf("failed ro read %x %x. reason:%v\n", twoBytes[0], twoBytes[1], err)
		return twoBytesData{}, err
	}
	return twoBytesData{num: int(read), high: twoBytes[0], low: twoBytes[1]}, nil
}

func readExactBytes(n int, port io.ReadWriteCloser) ([]byte, error) {
	b := make([]byte, 1, 1)
	result := make([]byte, n, n)
	for i := 0; i < n; i++ {
		for {
			readNum, err := port.Read(b)
			if err != nil {
				log.Printf("failed to read. reason:%v\n", err)
				return []byte{0}, err
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
	return result, nil
}

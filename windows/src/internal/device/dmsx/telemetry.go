package dmsx

import (
	"fmt"
	"math"
	"time"

	"encoding/hex"
	"encoding/binary"

	. "dronmotors/dmetrics/internal/device"
)

const (
	frameChannelText = 0
	frameChannelData = 1
)

type frame struct {
	Channel int
	Payload	[]byte
}

func decodeFrame(d []byte) (*frame, error) {
	const minlen int = 8

	if len(d) < minlen {
		return nil, errorf("frame length is to short")
	}

	chl := binary.LittleEndian.Uint32(d)
	d = d[4:]
	sum := binary.LittleEndian.Uint32(d[len(d)-4:])
	d = d[:len(d)-4]

	channel := (int)(chl >> 24)
	switch (channel) {
	case frameChannelText:
	case frameChannelData:
	default:
		return nil, errorf("frame channel %d is not supported", channel)
	}

	if ((int)(chl & 0xffff) != len(d)) {
		return nil, errorf("frame payload length mismatch")
	}

	validate := func(d []byte, sum uint32) bool {
		return true // TODO
	}

	if !validate(d, sum) {
		return nil, errorf("frame payload checksum mismatch")
	}

	return &frame{
		Channel: channel,
		Payload: d,
	}, nil
}

func (f frame) String() string {
	var s string

	switch f.Channel {
	case frameChannelText:
		s = fmt.Sprintf("FRAME-TEXT<%s>", string(f.Payload))
	case frameChannelData:
		s = fmt.Sprintf("FRAME-DATA<%s>", hex.EncodeToString(f.Payload))
	}

	return s
}

////////////////////////////////////////////////////////////////////////////////
// telemetry
////////////////////////////////////////////////////////////////////////////////

const (
	TlmIdxTs		= 0x1000 + iota
	TlmIdxLoad1
	TlmIdxLoad2
	TlmIdxLoad3
	TlmIdxTemp1
	TlmIdxTemp2
	TlmIdxTemp3
	TlmIdxBrake
	TlmIdxMotorI
	TlmIdxMotorU
	TlmIdxMotorP
	TlmIdxMotorRPM
	TlmIdxMotorThrottle
	TlmIdxGyroX
	TlmIdxGyroY
	TlmIdxGyroZ
)

type dataTelemetry struct {
	timeStamp time.Time

	Ts int32
	Load1 int32
	Load2 int32
	Load3 int32
	Temp1 float64
	Temp2 float64
	Temp3 float64
	Brake int32
	MotorI float64
	MotorU float64
	MotorP float64
	MotorRPM int32
	Throttle int32
	GyroX int32
	GyroY int32
	GyroZ int32

	Tag string
}

func (t dataTelemetry) String() string {
	return fmt.Sprintf(
		"%dµs r/min %d b/pos %d | %.02fA x %.02fV = %.02fW | %.02f°C %.02f°C | %d %d %d | %d %d %d",
		t.Throttle, t.MotorRPM, t.Brake, t.MotorI, t.MotorU, t.MotorP, t.Temp1, t.Temp2, t.Load1, t.Load2, t.Load3, t.GyroX, t.GyroY, t.GyroZ,
	)
}

func (t *dataTelemetry) decodeBytes(d []byte) {
	idx := binary.LittleEndian.Uint32(d[0:4])
	val := binary.LittleEndian.Uint32(d[4:8])
	switch idx {
	case TlmIdxTs:
		t.Ts = int32(val)
	case TlmIdxLoad1:
		t.Load1 = int32(val)
	case TlmIdxLoad2:
		t.Load2 = int32(val)
	case TlmIdxLoad3:
		t.Load3 = int32(val)
	case TlmIdxTemp1:
		t.Temp1 = float64(int32(val >> 2)) // MAX6675: mul 0.25
	case TlmIdxTemp2:
		t.Temp2 = float64(int32(val >> 2)) // MAX6675: mul 0.25
	case TlmIdxTemp3:
		t.Temp3 = float64(int32(val >> 2)) // MAX6675: mul 0.25
	case TlmIdxBrake:
		t.Brake = int32(val)
	case TlmIdxMotorI:
		t.MotorI = float64(int32(val))
	case TlmIdxMotorU:
		t.MotorU = float64(int32(val))
	case TlmIdxMotorP:
		t.MotorP = float64(int32(val))
	case TlmIdxMotorRPM:
		t.MotorRPM = int32(val)
	case TlmIdxMotorThrottle:
		t.Throttle = int32(val)
	case TlmIdxGyroX:
		t.GyroX = int32(val)
	case TlmIdxGyroY:
		t.GyroY = int32(val)
	case TlmIdxGyroZ:
		t.GyroZ = int32(val)
	}
}

func (t dataTelemetry) decode(f *frame) (Telemetry, error) {
	dt := dataTelemetry{}

	if len(f.Payload) < 8 {
		return nil, errorf("telemetry payload is too short")
	} else if (len(f.Payload) % 8) != 0 {
		return nil, errorf("telemetry payload length mismatch")
	}

	ver := binary.LittleEndian.Uint32(f.Payload[0:4]) // TLM_ID_V
	if (ver != 0x1000) {
		return nil, errorf("telemetry version %08x is not supported", ver)
	}

	dt.timeStamp = time.Now()
	for i := 0; i < len(f.Payload) / 8; i++ {
		dt.decodeBytes(f.Payload[i*8:(i+1)*8])
	}

	dt.fixup()

	return &dt, nil
}

func (t *dataTelemetry) fixup() {
	const currentLSB float64 = 0.01 // FIXME: INA236 manual
	t.MotorU = math.Round((t.MotorU * 0.0016) * 100) / 100 // to volts
	t.MotorI = math.Round((t.MotorI * currentLSB) * 100) / 100 // to ampers
	t.MotorP = math.Round((t.MotorP * (32 * currentLSB)) * 100) / 100 // to watts
}

func (t dataTelemetry) Id() string {
	return "Telemetry-1000"
}

func (t dataTelemetry) AsKeys() []string {
	return []string{
		"ts",
		"load1",
		"load2",
		"load3",
		"temp1",
		"temp2",
		"temp3",
		"motorI",
		"motorU",
		"motorP",
		"motorRPM",
		"throttle",
		"gyroX",
		"gyroY",
		"gyroZ",
		"tag",
	}
}

func (t dataTelemetry) AsValues() []string {
	return []string{
		fmt.Sprintf("%d", t.Ts),
		fmt.Sprintf("%d", t.Load1),
		fmt.Sprintf("%d", t.Load2),
		fmt.Sprintf("%d", t.Load3),
		fmt.Sprintf("%.02f", t.Temp1),
		fmt.Sprintf("%.02f", t.Temp2),
		fmt.Sprintf("%.02f", t.Temp3),
		fmt.Sprintf("%.02f", t.MotorI),
		fmt.Sprintf("%.02f", t.MotorU),
		fmt.Sprintf("%.02f", t.MotorP),
		fmt.Sprintf("%d", t.MotorRPM),
		fmt.Sprintf("%d", t.Throttle),
		fmt.Sprintf("%d", t.GyroX),
		fmt.Sprintf("%d", t.GyroY),
		fmt.Sprintf("%d", t.GyroZ),
		t.Tag,
	}
}

func (t dataTelemetry) TimeStamp() time.Time {
	return t.timeStamp
}

package node

import (
	"errors"
	"fmt"
	"log"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/db/genji"
	"github.com/simpleiot/simpleiot/modbus"
	"github.com/simpleiot/simpleiot/nats"
	"github.com/simpleiot/simpleiot/respreader"
	"go.bug.st/serial"
)

// ModbusManager manages state of modbus
type ModbusManager struct {
	db     *genji.Db
	nc     *natsgo.Conn
	busses map[string]*Modbus
}

// NewModbusManager creates a new modbus manager
func NewModbusManager(db *genji.Db, nc *natsgo.Conn) *ModbusManager {
	return &ModbusManager{
		db:     db,
		nc:     nc,
		busses: make(map[string]*Modbus),
	}
}

// Update queries DB for modbus nodes and synchronizes
// with internal structures and updates data
func (mm *ModbusManager) Update() error {
	rootID := mm.db.RootNodeID()
	nodes, err := mm.db.NodeChildren(rootID, data.NodeTypeModbus)
	if err != nil {
		return err
	}

	found := make(map[string]bool)

	for _, ne := range nodes {
		found[ne.ID] = true
		bus, ok := mm.busses[ne.ID]
		if !ok {
			var err error
			bus, err = NewModbus(mm.nc, &ne)
			if err != nil {
				log.Println("Error creating new modbus: ", err)
				continue
			}
			mm.busses[ne.ID] = bus
		}

		err := bus.CheckPort(&ne)
		if err != nil {
			log.Println("Error initializing modbus port: ",
				ne.ID, err)
			continue
		}

		ioNodes, err := mm.db.NodeChildren(ne.ID, data.NodeTypeModbusIO)
		if err != nil {
			log.Println("Error getting modbus IO nodes: ", err)
			continue
		}

		for _, ioNode := range ioNodes {
			io, err := NewModbusIO(bus.busType, &ioNode)
			if err != nil {
				log.Println("Error creating new modbus IO: ", err)
				continue
			}

			switch bus.busType {
			case data.PointValueServer:
				err := bus.ServerIO(io)
				if err != nil {
					log.Println("Modbus server IO error: ", err)
				}
			case data.PointValueClient:
				err := bus.ClientIO(io)
				if err != nil {
					log.Println("Modbus server IO error: ", err)
				}
			default:
				log.Println("unhandled modbus type: ", bus.busType)
			}
		}
	}

	// remove busses that have been deleted
	for id, bus := range mm.busses {
		_, ok := found[id]
		if !ok {
			// bus was deleted so close and clear it
			log.Println("Closing modbus on port: ", bus.portName)
			err := bus.port.Close()
			if err != nil {
				log.Println("Error closing modbus port: ", err)
			}

			delete(mm.busses, id)
		}
	}

	return nil
}

// Modbus describes a modbus bus
type Modbus struct {
	nc            *natsgo.Conn
	busType       string
	id            int // only used for server
	portName      string
	baud          int
	port          *respreader.ReadWriteCloser
	client        *modbus.Client
	server        *modbus.Server
	debugLevel    int
	ioInitialized map[string]bool
}

// NewModbus creates a new bus from a node
func NewModbus(nc *natsgo.Conn, node *data.NodeEdge) (*Modbus, error) {
	busType, ok := node.Points.Text("", data.PointTypeClientServer, 0)
	if !ok {
		return nil, errors.New("Must define modbus client/server")
	}
	portName, ok := node.Points.Text("", data.PointTypePort, 0)
	if !ok {
		return nil, errors.New("Must define modbus port name")
	}
	baud, ok := node.Points.Value("", data.PointTypeBaud, 0)
	if !ok {
		return nil, errors.New("Must define modbus baud")
	}

	return &Modbus{
		nc:            nc,
		busType:       busType,
		portName:      portName,
		baud:          int(baud),
		ioInitialized: make(map[string]bool),
	}, nil
}

// CheckPort verifies the serial port setup is correct for bus
func (bus *Modbus) CheckPort(node *data.NodeEdge) error {
	busType, ok := node.Points.Text("", data.PointTypeClientServer, 0)
	if !ok {
		return errors.New("Must define modbus client/server")
	}
	portName, ok := node.Points.Text("", data.PointTypePort, 0)
	if !ok {
		return errors.New("Must define modbus port name")
	}
	baud, ok := node.Points.Value("", data.PointTypeBaud, 0)
	if !ok {
		return errors.New("Must define modbus baud")
	}

	debugLevel, _ := node.Points.Value("", data.PointTypeDebug, 0)

	id := bus.id

	if busType == data.PointValueServer {
		idF, ok := node.Points.Value("", data.PointTypeID, 0)
		if !ok {
			return errors.New("Must define modbus ID for server bus")
		}

		id = int(idF)
	}

	if busType != bus.busType || portName != bus.portName ||
		int(baud) != bus.baud || id != bus.id ||
		int(debugLevel) != bus.debugLevel {
		// need to re-init port if it is open
		if bus.port != nil {
			bus.port.Close()
			bus.port = nil
		}

		bus.busType = busType
		bus.portName = portName
		bus.baud = int(baud)
		bus.id = id
		bus.debugLevel = int(debugLevel)
	}

	if bus.port == nil {
		log.Println("initializing modbus port: ", bus.portName)
		// need to init serial port
		mode := &serial.Mode{
			BaudRate: bus.baud,
		}

		port, err := serial.Open(bus.portName, mode)
		if err != nil {
			return err
		}

		bus.port = respreader.NewReadWriteCloser(port, time.Second*1, time.Millisecond*30)

		if bus.busType == data.PointValueServer {
			bus.client = nil
			bus.server = modbus.NewServer(byte(bus.id), bus.port)
			go bus.server.Listen(bus.debugLevel, func(err error) {
				log.Println("Modbus server error: ", err)
			}, func(changes []modbus.RegChange) {
				log.Println("Modbus reg change")
			})
		} else if bus.busType == data.PointValueClient {
			bus.server = nil
			bus.client = modbus.NewClient(bus.port, bus.debugLevel)
		}
	}

	return nil
}

// SendPoint sends a point over nats
func (bus *Modbus) SendPoint(nodeID, pointType string, value float64) error {
	// send the point
	p := data.Point{
		Type:  pointType,
		Value: value,
	}

	return nats.SendPoint(bus.nc, nodeID, &p, true)
}

// ReadReg reads an value from a reg (internal, not bus)
// This should only be used on server
func (bus *Modbus) ReadReg(io *ModbusIO) (float64, error) {
	var valueUnscaled float64
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		v, err := bus.server.Regs.ReadReg(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueUINT32:
		v, err := bus.server.Regs.ReadRegUint32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueINT32:
		v, err := bus.server.Regs.ReadRegInt32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	case data.PointValueFLOAT32:
		v, err := bus.server.Regs.ReadRegFloat32(io.address)
		if err != nil {
			return 0, err
		}
		valueUnscaled = float64(v)
	default:
		return 0, fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}
	return valueUnscaled*io.scale + io.offset, nil
}

// WriteReg writes an io value to a reg
// This should only be used on server
func (bus *Modbus) WriteReg(io *ModbusIO) error {
	unscaledValue := (io.value - io.offset) / io.scale
	bus.server.Regs.AddReg(io.address, regCount(io.modbusDataType))
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		bus.server.Regs.WriteReg(io.address, uint16(unscaledValue))
	case data.PointValueUINT32:
		bus.server.Regs.WriteRegUint32(io.address,
			uint32(unscaledValue))
	case data.PointValueINT32:
		bus.server.Regs.WriteRegInt32(io.address,
			int32(unscaledValue))
	case data.PointValueFLOAT32:
		bus.server.Regs.WriteRegFloat32(io.address,
			float32(unscaledValue))
	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}
	return nil
}

// WriteBusHoldingReg used to write register values to bus
// should only be used by client
func (bus *Modbus) WriteBusHoldingReg(io *ModbusIO) error {
	unscaledValue := (io.valueSet - io.offset) / io.scale
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), uint16(unscaledValue))
		if err != nil {
			return err
		}
	case data.PointValueUINT32:
		regs := modbus.Uint32ToRegs([]uint32{uint32(unscaledValue)})
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueINT32:
		regs := modbus.Int32ToRegs([]int32{int32(unscaledValue)})
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	case data.PointValueFLOAT32:
		regs := modbus.Float32ToRegs([]float32{float32(unscaledValue)})
		err := bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address), regs[0])
		if err != nil {
			return err
		}

		err = bus.client.WriteSingleReg(byte(io.id),
			uint16(io.address+1), regs[1])
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)

	}

	return nil
}

// ReadBusReg reads an io value from a reg from bus
// this function modifies io.value
func (bus *Modbus) ReadBusReg(io *ModbusIO) error {
	readFunc := bus.client.ReadHoldingRegs
	switch io.modbusType {
	case data.PointValueModbusHoldingRegister:
	case data.PointValueModbusInputRegister:
		readFunc = bus.client.ReadInputRegs
	default:
		return fmt.Errorf("ReadBusReg: unsupported modbus type: %v",
			io.modbusType)
	}
	var valueUnscaled float64
	switch io.modbusDataType {
	case data.PointValueUINT16, data.PointValueINT16:
		regs, err := readFunc(byte(io.id), uint16(io.address), 1)
		if err != nil {
			return err
		}
		if len(regs) < 1 {
			return errors.New("Did not receive enough data")
		}
		valueUnscaled = float64(regs[0])

	case data.PointValueUINT32:
		regs, err := readFunc(byte(io.id), uint16(io.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		v := modbus.RegsToUint32(regs)

		valueUnscaled = float64(v[0])

	case data.PointValueINT32:
		regs, err := readFunc(byte(io.id), uint16(io.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		v := modbus.RegsToInt32(regs)

		valueUnscaled = float64(v[0])

	case data.PointValueFLOAT32:
		regs, err := readFunc(byte(io.id), uint16(io.address), 2)
		if err != nil {
			return err
		}
		if len(regs) < 2 {
			return errors.New("Did not receive enough data")
		}
		valueUnscaled = float64(modbus.RegsToFloat32(regs)[0])

	default:
		return fmt.Errorf("unhandled data type: %v",
			io.modbusDataType)
	}

	io.value = valueUnscaled*io.scale + io.offset
	// send the point
	err := bus.SendPoint(io.nodeID, data.PointTypeValue, io.value)
	if err != nil {
		return err
	}

	return nil
}

// ReadBusBit is used to read coil of discrete input values from bus
// this function modifies io.value. This should only be called from client.
func (bus *Modbus) ReadBusBit(io *ModbusIO) error {
	readFunc := bus.client.ReadCoils
	switch io.modbusType {
	case data.PointValueModbusCoil:
	case data.PointValueModbusDiscreteInput:
		readFunc = bus.client.ReadDiscreteInputs
	default:
		return fmt.Errorf("ReadBusBit: unhandled modbusType: %v",
			io.modbusType)
	}
	bits, err := readFunc(byte(io.id), uint16(io.address), 1)
	if err != nil {
		return err
	}
	if len(bits) < 1 {
		return errors.New("Did not receive enough data")
	}
	io.value = data.BoolToFloat(bits[0])

	err = bus.SendPoint(io.nodeID, data.PointTypeValue, io.value)
	if err != nil {
		return err
	}

	return nil
}

// ClientIO processes an IO on a client bus
func (bus *Modbus) ClientIO(io *ModbusIO) error {
	// read value from remote device and update regs
	switch io.modbusType {
	case data.PointValueModbusCoil:
		err := bus.ReadBusBit(io)
		if err != nil {
			return err
		}

		if io.valueSet != io.value {
			vBool := data.FloatToBool(io.valueSet)
			// we need set the remote value
			err := bus.client.WriteSingleCoil(byte(io.id), uint16(io.address),
				vBool)

			if err != nil {
				return err
			}

			err = bus.SendPoint(io.nodeID, data.PointTypeValue, io.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusDiscreteInput:
		err := bus.ReadBusBit(io)
		if err != nil {
			return err
		}

	case data.PointValueModbusHoldingRegister:
		err := bus.ReadBusReg(io)
		if err != nil {
			return err
		}

		if io.valueSet != io.value {
			// we need set the remote value
			err := bus.WriteBusHoldingReg(io)

			if err != nil {
				return err
			}

			err = bus.SendPoint(io.nodeID, data.PointTypeValue, io.valueSet)
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		err := bus.ReadBusReg(io)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unhandled modbus io type: %v", io.modbusType)
	}

	return nil
}

// ServerIO processes an IO on a server bus
func (bus *Modbus) ServerIO(io *ModbusIO) error {
	// update regs with db value
	switch io.modbusType {
	case data.PointValueModbusDiscreteInput:
		bus.server.Regs.AddCoil(io.address)
		bus.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
	case data.PointValueModbusCoil:
		initialized := bus.ioInitialized[io.nodeID]
		if !initialized {
			bus.server.Regs.AddCoil(io.address)
			bus.server.Regs.WriteCoil(io.address, data.FloatToBool(io.value))
			bus.ioInitialized[io.nodeID] = true
		}
		regValue, err := bus.server.Regs.ReadCoil(io.address)
		if err != nil {
			return err
		}

		dbValue := data.FloatToBool(io.value)

		if regValue != dbValue {
			err = bus.SendPoint(io.nodeID, data.PointTypeValue, data.BoolToFloat(regValue))
			if err != nil {
				return err
			}
		}

	case data.PointValueModbusInputRegister:
		bus.WriteReg(io)

	case data.PointValueModbusHoldingRegister:
		// FIXME, how to handle case where address changes
		initialized := bus.ioInitialized[io.nodeID]
		if !initialized {
			bus.WriteReg(io)
			bus.ioInitialized[io.nodeID] = true
		}
		v, err := bus.ReadReg(io)
		if err != nil {
			return err
		}

		if io.value != v {
			err = bus.SendPoint(io.nodeID, data.PointTypeValue, v)
			if err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unhandled modbus io type: %v", io.modbusType)
	}

	return nil
}

// ModbusIO describes a modbus IO
type ModbusIO struct {
	nodeID         string
	description    string
	id             int
	address        int
	modbusType     string
	modbusDataType string
	scale          float64
	offset         float64
	value          float64
	valueSet       float64
}

// NewModbusIO Convert node to modbus IO
func NewModbusIO(busType string, node *data.NodeEdge) (*ModbusIO, error) {
	var ret ModbusIO
	var ok bool

	ret.nodeID = node.ID

	ret.id, ok = node.Points.ValueInt("", data.PointTypeID, 0)
	if busType == data.PointValueClient && !ok {
		return nil, errors.New("Must define modbus ID")
	}

	ret.address, ok = node.Points.ValueInt("", data.PointTypeAddress, 0)
	if !ok {
		return nil, errors.New("Must define modbus address")
	}
	ret.modbusType, ok = node.Points.Text("", data.PointTypeModbusIOType, 0)
	if !ok {
		return nil, errors.New("Must define modbus IO type")
	}

	if ret.modbusType == data.PointValueModbusInputRegister ||
		ret.modbusType == data.PointValueModbusHoldingRegister {
		ret.modbusDataType, ok = node.Points.Text("", data.PointTypeDataFormat, 0)
		if !ok {
			return nil, errors.New("Data format must be specified")
		}
		ret.scale, ok = node.Points.Value("", data.PointTypeScale, 0)
		if !ok {
			return nil, errors.New("Must define modbus scale")
		}
		ret.offset, ok = node.Points.Value("", data.PointTypeOffset, 0)
		if !ok {
			return nil, errors.New("Must define modbus offset")
		}
	}
	ret.value, _ = node.Points.Value("", data.PointTypeValue, 0)
	ret.valueSet, _ = node.Points.Value("", data.PointTypeValueSet, 0)

	return &ret, nil
}

func regCount(regType string) int {
	switch regType {
	case data.PointValueUINT16, data.PointValueINT16:
		return 1
	case data.PointValueUINT32, data.PointValueINT32,
		data.PointValueFLOAT32:
		return 2
	default:
		log.Println("regCount, unknown data type: ", regType)
		// be conservative
		return 2
	}
}

package data

// define common node and point types that have special meaning in
// the system.
const (
	// general point types
	PointTypeDescription string = "description"
	PointTypeScale              = "scale"
	PointTypeOffset             = "offset"
	PointTypeUnits              = "units"
	PointTypeValue              = "value"
	PointTypeValueSet           = "valueSet"
	PointTypeID                 = "id"
	PointTypeAddress            = "address"
	PointTypeDebug              = "debug"
	PointTypeInitialized        = "initialized"

	// An device node describes an phyical device -- it may be the
	// cloud server, gateway, etc
	NodeTypeDevice                = "device"
	PointTypeCmdPending           = "cmdPending"
	PointTypeSwUpdateState        = "swUpdateState"
	PointTypeStartApp             = "startApp"
	PointTypeStartSystem          = "startSystem"
	PointTypeUpdateOS             = "updateOS"
	PointTypeUpdateApp            = "updateApp"
	PointTypeSysState             = "sysState"
	PointTypeSwUpdateRunning      = "swUpdateRunning"
	PointTypeSwUpdateError        = "swUpdateError"
	PointTypeSwUpdatePercComplete = "swUpdatePercComplete"
	PointTypeOSVersion            = "osVersion"
	PointTypeAppVersion           = "appVersion"
	PointTypeHwVersion            = "hwVersion"

	// user node describes a system user and is used to control
	// access to the system (typically through web UI)
	NodeTypeUser       = "user"
	PointTypeFirstName = "firstName"
	PointTypeLastName  = "lastName"
	PointTypePhone     = "phone"
	PointTypeEmail     = "email"
	PointTypePass      = "pass"

	// modbus nodes
	// in modbus land, terminology is a big backwards, client is master,
	// and server is slave.
	NodeTypeModbus        = "modbus"
	PointTypeClientServer = "clientServer"
	PointValueClient      = "client"
	PointValueServer      = "server"
	PointTypePort         = "port"
	PointTypeBaud         = "baud"

	NodeTypeModbusIO = "modbusIo"

	PointTypeModbusIOType           = "modbusIoType"
	PointValueModbusDiscreteInput   = "modbusDiscreteInput"
	PointValueModbusCoil            = "modbusCoil"
	PointValueModbusInputRegister   = "modbusInputRegister"
	PointValueModbusHoldingRegister = "modbusHoldingRegister"

	PointTypeDataFormat = "dataFormat"
	PointValueUINT16    = "uint16"
	PointValueINT16     = "int16"
	PointValueUINT32    = "uint32"
	PointValueINT32     = "int32"
	PointValueFLOAT32   = "float32"

	// A group node is used to group users and devices
	// or generally to add structure to the node graph.
	NodeTypeGroup = "group"

	// a rule node describes a rule that may run on the system
	NodeTypeRule          = "rule"
	NodeTypeRuleCondition = "ruleCondition"
	NodeTypeRuleAction    = "ruleAction"

	// Transient points that are used for notifications, etc.
	// These points are not stored in the state state of any node,
	// but are recorded in the time series database to record history.
	PointMsgAll  = "msgAll"
	PointMsgUser = "msgUser"
)

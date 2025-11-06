package cmd

import (
	"reflect"
	"sync"

	"google.golang.org/protobuf/proto"
)

var sharedCmdProtoMap *CmdProtoMap
var cmdProtoMapOnce sync.Once

type CmdProtoMap struct {
	protoObjCmdIdMap map[reflect.Type]uint32

	cmdIdCmdNameMap map[uint32]string
	cmdNameCmdIdMap map[string]uint32

	cmdIdProtoObjMap map[uint32]reflect.Type
}

func Get() *CmdProtoMap {
	cmdProtoMapOnce.Do(func() {
		sharedCmdProtoMap = NewCmdProtoMap()
	})
	return sharedCmdProtoMap
}

func NewCmdProtoMap() (r *CmdProtoMap) {
	r = new(CmdProtoMap)
	r.protoObjCmdIdMap = make(map[reflect.Type]uint32)
	r.cmdIdCmdNameMap = make(map[uint32]string)
	r.cmdNameCmdIdMap = make(map[string]uint32)
	r.cmdIdProtoObjMap = make(map[uint32]reflect.Type)
	r.registerAllMessage()

	return r
}

func (c *CmdProtoMap) regMsg(cmdId uint32, protoObjNewFunc func() any) {
	protoObj := protoObjNewFunc().(proto.Message)
	refType := reflect.TypeOf(protoObj)
	// protoObj -> cmdId
	c.protoObjCmdIdMap[refType] = cmdId
	cmdName := refType.Elem().Name()
	// cmdId -> cmdName
	c.cmdIdCmdNameMap[cmdId] = cmdName
	// cmdName -> cmdId
	c.cmdNameCmdIdMap[cmdName] = cmdId

	c.cmdIdProtoObjMap[cmdId] = refType
}

// 反射方法

func (c *CmdProtoMap) GetProtoObjByCmdId(cmdId uint32) proto.Message {
	refType, exist := c.cmdIdProtoObjMap[cmdId]
	if !exist {
		return nil
	}
	protoObjInst := reflect.New(refType.Elem())
	protoObj := protoObjInst.Interface().(proto.Message)
	return protoObj
}

func (c *CmdProtoMap) GetCmdIdByProtoObj(protoObj proto.Message) uint32 {
	cmdId, exist := c.protoObjCmdIdMap[reflect.TypeOf(protoObj)]
	if !exist {
		return 0
	}
	return cmdId
}

func (c *CmdProtoMap) GetCmdNameByCmdId(cmdId uint32) string {
	cmdName, exist := c.cmdIdCmdNameMap[cmdId]
	if !exist {
		return ""
	}
	return cmdName
}

func (c *CmdProtoMap) GetCmdIdByCmdName(cmdName string) uint32 {
	cmdId, exist := c.cmdNameCmdIdMap[cmdName]
	if !exist {
		return 0
	}
	return cmdId
}

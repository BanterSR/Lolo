package main

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/proto"

	"gucooing/lolo/protocol/cmd"
)

func GetProtoNameById(id uint32) string {
	return cmd.Get().GetCmdNameByCmdId(id)
}

func parseProto(id uint32, data []byte) (proto.Message, error) {
	ojb := cmd.Get().GetProtoObjByCmdId(id)
	if ojb == nil {
		return nil, errors.New("Unknown proto ")
	}
	err := proto.Unmarshal(data, ojb)

	return ojb, err
}

func parseProtoToJson(id uint32, data []byte) (string, error) {
	ojb, err := parseProto(id, data)
	if err != nil {
		return "", err
	}

	marshalJSON, err := json.Marshal(ojb)
	if err != nil {
		return "", err
	}

	return string(marshalJSON), nil
}

func parseProtoToInterface(id uint32, data []byte) (*interface{}, error) {
	object, err := parseProtoToJson(id, data)
	if err != nil {
		return nil, err
	}

	var result *interface{}
	err = json.Unmarshal([]byte(object), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

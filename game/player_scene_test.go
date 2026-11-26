package game

import (
	"testing"

	"gucooing/lolo/protocol/proto"
)

func TestSceneRecorderValidationRejectsInvalidMovement(t *testing.T) {
	if validSceneRecorderData(nil) {
		t.Fatal("nil recorder data must be rejected")
	}
	for _, position := range []*proto.Vector3{
		{X: 1_000_001},
		{Y: -1_000_001},
		{Z: 1_000_001},
	} {
		data := &proto.PlayerRecorderData{
			CharRecorderDataLst: []*proto.CharacterRecorderData{{Pos: position}},
		}
		if validSceneRecorderData(data) {
			t.Fatalf("invalid position %+v must be rejected", position)
		}
	}
}

func TestSceneRecorderValidationAcceptsBoundedMovement(t *testing.T) {
	data := &proto.PlayerRecorderData{
		CharRecorderDataLst: []*proto.CharacterRecorderData{
			{CharId: 1, Pos: &proto.Vector3{X: 1, Y: 2, Z: 3}, Rot: &proto.Vector3{Y: 180}},
		},
		PetRecordDataLst: []*proto.CharacterRecorderData{{CharId: 2}},
	}
	if !validSceneRecorderData(data) {
		t.Fatal("valid movement must be accepted")
	}
	data.CharRecorderDataLst = make([]*proto.CharacterRecorderData, 17)
	if validSceneRecorderData(data) {
		t.Fatal("oversized character recorder list must be rejected")
	}
}

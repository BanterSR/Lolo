package model

import (
	"testing"

	"gucooing/lolo/protocol/proto"
)

func TestQuestDetailRemovesCorruptLegacyEntries(t *testing.T) {
	player := &Player{Quest: &QuestModel{QuestMap: map[uint32]*proto.Quest{
		0:      {},
		100011: nil,
	}}}
	detail := player.GetQuestDetail()
	if len(detail.GetQuests()) != 0 {
		t.Fatalf("corrupt quests were sent to the client: %+v", detail.GetQuests())
	}
	if len(player.GetQuestModel().GetQuestMap()) != 0 {
		t.Fatal("corrupt quest entries were not removed from the save")
	}
}

func TestEmptyLifeBlobMigratesToFourLevelOneStates(t *testing.T) {
	player := &Player{Life: &LifeModel{}}
	life := player.GetLifeModel()
	for lifeType := proto.LifeType_LIFE_TYPE_Cooking; lifeType <= proto.LifeType_LIFE_TYPE_Handicraft; lifeType++ {
		state := life.GetState(lifeType)
		if state.Level != 1 || state.Skills == nil || state.Achievements == nil {
			t.Fatalf("life type %d was not migrated: %+v", lifeType, state)
		}
	}
}

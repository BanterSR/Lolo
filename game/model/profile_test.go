package model

import (
	"testing"

	"github.com/bytedance/sonic"
)

func TestProfileModelPersistsClaimsAndCooldowns(t *testing.T) {
	profile := DefaultProfileModel()
	profile.PhoneCase = 7000
	profile.EnergyLastRecoverAt = 100
	profile.EnergyBuyDay = 20
	profile.EnergyBuyNum = 2
	profile.MarkPlayerLevelReward(5)
	profile.MarkWorldLevelUnlocked(1)
	profile.CompletedAchievements[10] = true
	profile.CompletedFlags[20] = true
	profile.LastPopEmojiAt = 200
	profile.LastClapAt = 300
	profile.LastThrowDiceAt = 400

	bin, err := sonic.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var restored ProfileModel
	if err := sonic.Unmarshal(bin, &restored); err != nil {
		t.Fatal(err)
	}
	restored.EnsureMaps()
	if !restored.HasPlayerLevelReward(5) || !restored.IsWorldLevelUnlocked(1) ||
		!restored.CompletedAchievements[10] || !restored.CompletedFlags[20] ||
		restored.PhoneCase != 7000 || restored.EnergyBuyNum != 2 ||
		restored.LastPopEmojiAt != 200 || restored.LastClapAt != 300 || restored.LastThrowDiceAt != 400 {
		t.Fatalf("profile state did not round trip: %+v", restored)
	}
	if len(restored.RewardedPlayerLevels()) != 1 || restored.RewardedPlayerLevels()[0] != 5 {
		t.Fatalf("unexpected rewarded levels: %v", restored.RewardedPlayerLevels())
	}
}

func TestProfileModelRejectsZeroIdentifiers(t *testing.T) {
	profile := DefaultProfileModel()
	profile.MarkPlayerLevelReward(0)
	profile.MarkWorldLevelUnlocked(0)
	if len(profile.PlayerLevelRewards) != 0 || len(profile.UnlockedWorldLevels) != 0 {
		t.Fatal("zero identifiers must not create claims")
	}
}

package model

import "gucooing/lolo/protocol/proto"

// RestaurantModel 保存餐厅小游戏的按活动进度。所有写操作由 game 主循环串行执行。
type RestaurantModel struct {
	Activities map[uint32]*RestaurantActivityState `json:"activities,omitempty"`
}

type RestaurantActivityState struct {
	Day              int32                              `json:"day,omitempty"`
	CurrentDungeonID uint32                             `json:"currentDungeonId,omitempty"`
	Dungeons         map[uint32]*RestaurantDungeonState `json:"dungeons,omitempty"`
	Quests           map[uint32]*RestaurantQuestState   `json:"quests,omitempty"`
}

type RestaurantDungeonState struct {
	Grids      map[uint32]uint32                    `json:"grids,omitempty"`
	Customers  []RestaurantCustomerState            `json:"customers,omitempty"`
	Cleared    bool                                 `json:"cleared,omitempty"`
	Operations map[string]RestaurantOperationRecord `json:"operations,omitempty"`
}

type RestaurantOperationRecord struct {
	Day   int32             `json:"day,omitempty"`
	Grids map[uint32]uint32 `json:"grids,omitempty"`
}

type RestaurantCustomerState struct {
	Food   uint32 `json:"food,omitempty"`
	Served uint32 `json:"served,omitempty"`
	Done   bool   `json:"done,omitempty"`
}

type RestaurantQuestState struct {
	Status   proto.QuestStatus `json:"status,omitempty"`
	Progress uint32            `json:"progress,omitempty"`
	Rewarded bool              `json:"rewarded,omitempty"`
	Day      int32             `json:"day,omitempty"`
}

// PetModel 保存CaptureReq缺少请求号时的捕获尝试凭据，避免重放造成重复宠物或重复扣料。
type PetModel struct {
	Attempts map[string]PetCaptureAttempt `json:"attempts,omitempty"`
}

type PetCaptureAttempt struct {
	Success       bool   `json:"success,omitempty"`
	PetInstanceID uint32 `json:"petInstanceId,omitempty"`
}

func DefaultRestaurantModel() *RestaurantModel {
	return &RestaurantModel{Activities: make(map[uint32]*RestaurantActivityState)}
}

func DefaultPetModel() *PetModel {
	return &PetModel{Attempts: make(map[string]PetCaptureAttempt)}
}

func (s *Player) GetRestaurantModel() *RestaurantModel {
	if s.Restaurant == nil {
		s.Restaurant = DefaultRestaurantModel()
	}
	if s.Restaurant.Activities == nil {
		s.Restaurant.Activities = make(map[uint32]*RestaurantActivityState)
	}
	return s.Restaurant
}

func (s *Player) GetPetModel() *PetModel {
	if s.Pet == nil {
		s.Pet = DefaultPetModel()
	}
	if s.Pet.Attempts == nil {
		s.Pet.Attempts = make(map[string]PetCaptureAttempt)
	}
	return s.Pet
}

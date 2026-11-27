package gdconf

import (
	"gucooing/lolo/protocol/excel"
)

type Story struct {
	all           *excel.AllStoryDatas
	StoryChapters map[uint32]*StoryChapter // 故事章节
}

type StoryChapter struct {
	Config    *excel.StoryChapterConfigure
	StoryList map[uint32]*excel.StoryConfigure
}

func (g *GameConfig) loadStory() {
	info := &Story{
		all:           new(excel.AllStoryDatas),
		StoryChapters: make(map[uint32]*StoryChapter),
	}
	g.Excel.Story = info
	name := "Story.json"
	ReadJson(g.excelPath, name, &info.all)

	for _, v := range info.all.GetStoryChapter().GetDatas() {
		storyChapter := &StoryChapter{
			Config:    v,
			StoryList: make(map[uint32]*excel.StoryConfigure),
		}
		info.StoryChapters[uint32(v.ID)] = storyChapter
		for _, storyId := range v.GetStoryList() {
			for _, v2 := range info.all.GetStory().GetDatas() {
				if storyId == v2.ID {
					storyChapter.StoryList[uint32(v2.ID)] = v2
					break
				}
			}
		}
	}
}

func GetStoryChapters() map[uint32]*StoryChapter {
	if cc == nil || cc.Excel == nil || cc.Excel.Story == nil {
		return nil
	}
	return cc.Excel.Story.StoryChapters
}

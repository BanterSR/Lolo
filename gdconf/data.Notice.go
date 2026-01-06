package gdconf

type Notice struct {
	NoticeList []*NoticeInfo
}

type NoticeInfo struct {
	NoticeId         int        `json:"noticeId"`
	Title            string     `json:"title"`
	Content          []*Content `json:"content"`
	StartTime        int        `json:"startTime"`
	EndTime          int64      `json:"endTime"`
	LastUpdTime      int        `json:"lastUpdTime"`
	OrderId          int        `json:"orderId"`
	BeforeLoginPopup bool       `json:"beforeLoginPopup"`
	OriNoticeId      int        `json:"oriNoticeId"`
}

type Content struct {
	OrderId     int    `json:"orderId"`
	Text        string `json:"text"`
	ImageUrl    string `json:"imageUrl"`
	ImageHeight int    `json:"imageHeight"`
}

func (g *GameConfig) loadNotice() {
	g.Data.Notice = new(Notice)
	ReadJson(g.dataPath, "NoticeList.json", &g.Data.Notice.NoticeList)
}

func GetNoticeList() []*NoticeInfo {
	return cc.Data.Notice.NoticeList
}

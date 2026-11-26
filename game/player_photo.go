package game

import (
	"strconv"
	"strings"
	"time"

	"gucooing/lolo/game/model"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/proto"
)

func (g *Game) PhotoShareSearch(s *model.Player, msg *alg.GameMsg) {
	req, ok := msg.Body.(*proto.PhotoShareSearchReq)
	rsp := &proto.PhotoShareSearchRsp{
		Status:           proto.StatusCode_StatusCode_OK,
		Photos:           make([]*proto.PhotoPreviewInfo, 0),
		ActivityPhotos:   make([]*proto.PhotoPreviewInfo, 0),
		TodayUploadLimit: 100,
		UploadMaxNum:     100,
	}
	defer g.send(s, msg.PacketId, rsp)
	if !ok || req == nil {
		rsp.Status = proto.StatusCode_StatusCode_BAD_REQ
		return
	}

	archive := s.GetArchive()
	archiveMap := archive.GetArchiveMap()
	if maxUpload, err := strconv.ParseUint(archiveMap["photo:uploadMax"], 10, 32); err == nil && maxUpload > 0 {
		rsp.UploadMaxNum = uint32(maxUpload)
	}
	codes := sortedPhotoCodes(archiveMap)
	if req.SearchType == proto.PhotoSearchType_PhotoSearchType_MY_FAVORITE {
		// 收藏关系尚未建立，返回空列表而不是伪造收藏。
		codes = nil
	}
	startOfDay := time.Now().Truncate(24 * time.Hour).Unix()
	keyword := strings.ToLower(strings.TrimSpace(req.Text))

	matched := make([]int64, 0, len(codes))
	for _, code := range codes {
		prefix := g.photoArchivePrefix() + strconv.FormatInt(code, 10) + ":"
		uploadTime, _ := strconv.ParseInt(archiveMap[prefix+"time"], 10, 64)
		activity := archiveMap[prefix+"activity"] == "1"

		switch req.SearchType {
		case proto.PhotoSearchType_PhotoSearchType_TODAY:
			if uploadTime < startOfDay {
				continue
			}
		case proto.PhotoSearchType_PhotoSearchType_Activity,
			proto.PhotoSearchType_PhotoSearchType_Activity_Rank:
			if !activity {
				continue
			}
		}

		switch req.TextType {
		case proto.PhotoSearchTextType_PhotoSearchTextType_CODE:
			if strconv.FormatInt(code, 10) != strings.TrimSpace(req.Text) {
				continue
			}
		case proto.PhotoSearchTextType_PhotoSearchTextType_UID:
			if strconv.FormatUint(uint64(s.UserId), 10) != strings.TrimSpace(req.Text) {
				continue
			}
		case proto.PhotoSearchTextType_PhotoSearchTextType_TEXT:
			if keyword != "" {
				fields := []string{
					archiveMap[prefix+"title"],
					archiveMap[prefix+"tag1"],
					archiveMap[prefix+"tag2"],
					archiveMap[prefix+"tag3"],
				}
				hit := false
				for _, field := range fields {
					if strings.Contains(strings.ToLower(field), keyword) {
						hit = true
						break
					}
				}
				if !hit {
					continue
				}
			}
		}
		matched = append(matched, code)
	}

	rsp.TotalNum = uint32(len(matched))
	start := int(req.StartIndex)
	end := int(req.EndIndex)
	if start < 0 || start > len(matched) {
		start = len(matched)
	}
	if end <= start || end > len(matched) {
		end = len(matched)
	}
	for _, code := range matched[start:end] {
		prefix := g.photoArchivePrefix() + strconv.FormatInt(code, 10) + ":"
		activityID := uint32(0)
		if archiveMap[prefix+"activity"] == "1" {
			activityID = 1
		}
		alg.AddList(&rsp.Photos, &proto.PhotoPreviewInfo{
			Code:       code,
			PlayerId:   s.UserId,
			PlayerName: s.NickName,
			Title:      archiveMap[prefix+"title"],
			ActivityId: activityID,
			Status:     proto.AuditStatus_AuditStatus_SUCCESS,
		})
	}
	rsp.EndIndex = uint32(end)
	if req.SearchType == proto.PhotoSearchType_PhotoSearchType_Activity ||
		req.SearchType == proto.PhotoSearchType_PhotoSearchType_Activity_Rank {
		rsp.ActivityPhotos = rsp.Photos
		rsp.Photos = make([]*proto.PhotoPreviewInfo, 0)
	}
}

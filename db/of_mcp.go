package db

// 只读离线查询辅助方法,供 mcp/tools 查询玩家数据使用(不触碰在线内存数据,避免并发竞争)

// CountGameBasic 统计已注册玩家数量
func CountGameBasic() (int64, error) {
	var count int64
	err := db.Model(&OFGameBasic{}).Count(&count).Error
	return count, err
}

// ListGameBasic 分页拉取玩家基础信息(按 UserId 升序)
func ListGameBasic(limit, offset int) ([]*OFGameBasic, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var list []*OFGameBasic
	err := db.Model(&OFGameBasic{}).
		Order("user_id ASC").
		Limit(limit).
		Offset(offset).
		Find(&list).Error
	return list, err
}

// SearchGameBasicByNickName 按昵称模糊搜索玩家(大小写不敏感由数据库决定)
func SearchGameBasicByNickName(name string, limit int) ([]*OFGameBasic, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var list []*OFGameBasic
	err := db.Model(&OFGameBasic{}).
		Where("nick_name LIKE ?", "%"+name+"%").
		Order("user_id ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

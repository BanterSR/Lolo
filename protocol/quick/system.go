package quick

type SystemInitRequest struct {
	Imsi         string `json:"imsi"`
	OsLang       string `json:"os_lang"`
	ScreenWidth  string `json:"screen_width"`
	OsName       string `json:"os_name"`
	ScreenHeight string `json:"screen_height"`
	AuthToken    string `json:"auth_token"`
	GameVer      string `json:"game_ver"`
	DevImei      string `json:"dev_imei"`
	ChanelCkey   string `json:"chanel_ckey"`
	DevName      string `json:"dev_name"`
	ProductCkey  string `json:"product_ckey"`
	Platform     int    `json:"platform"`
	TimeStamp    string `json:"time_stamp"`
	Oaid         string `json:"oaid"`
	DeviceId     string `json:"device_id"`
	PushToken    string `json:"push_token"`
	CountryCode  string `json:"country_code"`
	OsVer        string `json:"os_ver"`
	SdkVer       string `json:"sdk_ver"`
}

type SystemInitResultV2 struct {
	OrigPwd      int       `json:"origPwd"`
	ClientIp     string    `json:"clientIp"`
	PtConfig     *PtConfig `json:"pt_config"`
	PtVer        *PtVer    `json:"pt_ver"`
	RealnameNode string    `json:"realname_node"`
}

type SystemInitResultV1 struct {
	OrigPwd       int          `json:"origPwd"`
	ClientIp      string       `json:"clientIp"`
	ProductConfig *PtConfig    `json:"productConfig"`
	Version       *PtVer       `json:"version"`
	RealnameNode  string       `json:"realNameNode"`
	PayTypes      []*PayType   `json:"payTypes"`
	UseEWallet    string       `json:"useEWallet"`
	AppAuthInfo   *AppAuthInfo `json:"appAuthInfo"`
	UcentUrl      string       `json:"ucentUrl"`
	SubUserRole   int          `json:"subUserRole"`
}

type PayType struct {
	PayTypeId string  `json:"payTypeId"`
	Sort      string  `json:"sort"`
	BackupGid string  `json:"backupGid"`
	PayName   string  `json:"payName"`
	Rebate    *Rebate `json:"rebate"`
}

type Rebate struct {
	Rate       int           `json:"rate"`
	Rateval    string        `json:"rateval"`
	RateConfig []interface{} `json:"rateConfig"`
}

type AppAuthInfo struct {
	AppLogo       string `json:"appLogo"`
	AppPackage    string `json:"appPackage"`
	Theme         string `json:"theme"`
	DefaultAvatar string `json:"defaultAvatar"`
}

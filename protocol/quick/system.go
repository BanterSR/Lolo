package quick

type SystemInitRequest struct {
	ScreenWidth    string `json:"screenWidth"`
	Latitude       string `json:"latitude"`
	AuthToken      string `json:"authToken"`
	DeviceName     string `json:"deviceName"`
	DeviceId       string `json:"deviceId"`
	Platform       int    `json:"platform"`
	OsVersion      string `json:"osVersion"`
	CountryCode    string `json:"countryCode"`
	AndId          string `json:"andId"`
	Dpi            string `json:"dpi"`
	OsLanguage     string `json:"osLanguage"`
	Oaid           string `json:"oaid"`
	Longitude      string `json:"longitude"`
	ChannelCode    string `json:"channelCode"`
	NetType        string `json:"netType"`
	ScreenHeight   string `json:"screenHeight"`
	ClientLang     string `json:"clientLang"`
	Ismobiledevice string `json:"ismobiledevice"`
	Flashversion   string `json:"flashversion"`
	OsName         string `json:"osName"`
	PushToken      string `json:"pushToken"`
	ProductCode    string `json:"productCode"`
	Isjailbroken   string `json:"isjailbroken"`
	Javasupport    string `json:"javasupport"`
	Imei           string `json:"imei"`
	GameVersion    int    `json:"gameVersion"`
	SignMd5        string `json:"signMd5"`
	SdkVersion     int    `json:"sdkVersion"`
	Time           int64  `json:"time"`
	Defaultbrowser string `json:"defaultbrowser"`
	IsEmt          string `json:"isEmt"`
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

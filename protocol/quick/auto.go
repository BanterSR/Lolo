package quick

type LoginByNameRequest struct {
	AuthToken   string `json:"authToken"`
	ClientLang  string `json:"clientLang"`
	DeviceId    string `json:"deviceId"`
	Platform    int    `json:"platform"`
	Password    string `json:"password"`
	ProductCode string `json:"productCode"`
	AndId       string `json:"andId"`
	GameVersion int    `json:"gameVersion"`
	SignMd5     string `json:"signMd5"`
	Imei        string `json:"imei"`
	SdkVersion  int    `json:"sdkVersion"`
	Time        int64  `json:"time"`
	Oaid        string `json:"oaid"`
	IsEmt       string `json:"isEmt"`
	Username    string `json:"username"`
	ChannelCode string `json:"channelCode"`
}

type AutoLoginRequest struct {
	ProductCkey string `json:"product_ckey"`
	DeviceId    string `json:"device_id"`
	Platform    int    `json:"platform"`
	TimeStamp   string `json:"time_stamp"`
	ChanelCkey  string `json:"chanel_ckey"`
	SdkVer      string `json:"sdk_ver"`
	GameVer     string `json:"game_ver"`
	AuthToken   string `json:"auth_token"`
	Oaid        string `json:"oaid"`
}

type LoginResultV1 struct {
	AuthToken     string      `json:"authToken"`
	UserData      *UserDataV1 `json:"userData"`
	CheckRealName int         `json:"checkRealName"`
	IsAdult       bool        `json:"isAdult"`
	UAge          int         `json:"uAge"`
	CkPlayTime    int         `json:"ckPlayTime"`
	GuestRealName int         `json:"guestRealName"`
	Id            int         `json:"id"`
	Message       string      `json:"message"`
	ExtInfo       *ExtInfo    `json:"extInfo"`
}

type LoginResultV2 struct {
	ExtInfo       *ExtInfo    `json:"extInfo"`
	IsAdult       bool        `json:"isAdult"`
	UAge          int         `json:"uAge"`
	CkPlayTime    int         `json:"ckPlayTime"`
	GuestRealName int         `json:"guestRealName"`
	Id            int         `json:"id"`
	Message       string      `json:"message"`
	AuthToken     string      `json:"auth_token"`
	UserData      *UserDataV2 `json:"user_data"`
	CheckRealname int         `json:"check_realname"`
}

type UserExtraInfoRequest struct {
	AuthToken   string `json:"authToken"`
	ClientLang  string `json:"clientLang"`
	DeviceId    string `json:"deviceId"`
	Platform    int    `json:"platform"`
	Uid         string `json:"uid"`
	ProductCode string `json:"productCode"`
	AndId       string `json:"andId"`
	GameVersion int    `json:"gameVersion"`
	SignMd5     string `json:"signMd5"`
	Imei        string `json:"imei"`
	SdkVersion  int    `json:"sdkVersion"`
	Time        int64  `json:"time"`
	Oaid        string `json:"oaid"`
	IsEmt       string `json:"isEmt"`
	ChannelCode string `json:"channelCode"`
}

type UserExtraInfo struct {
	IsBindPhone   int       `json:"isBindPhone"`
	NickName      string    `json:"nickName"`
	Phone         string    `json:"phone"`
	SexType       SexType   `json:"sexType"`
	RegType       string    `json:"regType"`
	LastLoginTime string    `json:"lastLoginTime"`
	FcmShowTips   int       `json:"fcmShowTips"`
	IsAdult       int       `json:"isAdult"`
	Timeleft      int       `json:"timeleft"`
	BindInfo      *BindInfo `json:"bindInfo"`
}

type BindInfo struct {
	BindWX    *BindQd `json:"bindWX"`
	BindQQ    *BindQd `json:"bindQQ"`
	BindApple *BindQd `json:"bindApple"`
}

type BindQd struct {
	IsBind int    `json:"isBind"`
	Bid    int    `json:"bid"`
	Buid   string `json:"buid"`
}

type AsyUonlineRequest struct {
	AuthToken   string `json:"authToken"`
	ClientLang  string `json:"clientLang"`
	DeviceId    string `json:"deviceId"`
	Platform    int    `json:"platform"`
	ProductCode string `json:"productCode"`
	AndId       string `json:"andId"`
	GameVersion int    `json:"gameVersion"`
	SignMd5     string `json:"signMd5"`
	Imei        string `json:"imei"`
	SdkVersion  int    `json:"sdkVersion"`
	Time        int64  `json:"time"`
	TimeLeft    int    `json:"timeLeft"`
	Oaid        string `json:"oaid"`
	IsEmt       string `json:"isEmt"`
	ChannelCode string `json:"channelCode"`
}

type CheckLoginRequest struct {
	ChannelCode  int    `json:"channel_code"`
	Platform     int    `json:"platform"`
	DeviceId     int    `json:"device_id"`
	DeviceOs     int    `json:"device_os"`
	DeviceOsV    int    `json:"device_os_v"`
	DeviceName   int    `json:"device_name"`
	Imei         int    `json:"imei"`
	Mid          int    `json:"mid"`
	GameName     string `json:"gameName"`
	SdkV         string `json:"sdk_v"`
	SdkSubV      string `json:"sdk_sub_v"`
	ChannelSdkV  string `json:"channel_sdk_v"`
	ProductV     int    `json:"product_v"`
	SessionId    string `json:"session_id"`
	Debug        int    `json:"debug"`
	ScreenHeight int    `json:"screen_height"`
	ScreenWidth  int    `json:"screen_width"`
	Dpi          int    `json:"dpi"`
	NetType      int    `json:"net_type"`
	Time         int64  `json:"time"`
	PackageName  string `json:"package_name"`
	SignMd5      string `json:"sign_md5"`
	Oaid         string `json:"oaid"`
	Uid          string `json:"uid"`
	Certificates string `json:"certificates"`
	SignIssuer   string `json:"signIssuer"`
	UserName     string `json:"user_name"`
	SignSubject  string `json:"signSubject"`
	SignSha1     string `json:"sign_sha1"`
	SignMd51     string `json:"signMd5"`
	Androidid    string `json:"androidid"`
	Token        string `json:"token"`
	CheckTime    string `json:"check_time"`
}

type CheckLoginResponse struct {
	Uid       string `json:"uid"`
	UserName  string `json:"user_name"`
	Token     string `json:"token"`
	UserToken string `json:"user_token"`
}

type CheckSdkTokenRequest struct {
	Token string `json:"token"`
	UID   string `json:"uid"`
}

type CheckSdkTokenResponse struct {
	Code int    `json:"code"`
	Uid  string `json:"uid"`
}

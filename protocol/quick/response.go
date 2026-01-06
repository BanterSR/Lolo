package quick

type Response struct {
	Result  bool        `json:"result"`
	Status  bool        `json:"status"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Error   *Error      `json:"error"`
}

func NewResponse() *Response {
	return &Response{
		Result:  true,
		Status:  true,
		Data:    nil,
		Message: "",
		Error:   &Error{},
	}
}

type Error struct {
	Id      int    `json:"id"`
	Message string `json:"message"`
}

func (r *Response) SetError(e string) {
	r.Result = false
	r.Error.Id = 1
	r.Error.Message = e
}

func (r *Response) SetData(data interface{}) {
	r.Data = data
}

type SexType int

const (
	GENDER_UNDEFINE SexType = 0
	GENDER_MALE     SexType = 1
	GENDER_FEMALE   SexType = 2
)

type ExtInfo struct {
	OauthType   int    `json:"oauthType"`
	OauthId     string `json:"oauthId"`
	AccessToken string `json:"access_token"`
}

type UserDataV1 struct {
	Uid       string `json:"uid"`
	Username  string `json:"username"`
	Token     string `json:"token"`
	IsGuest   int    `json:"isGuest"`
	IsMbUser  int    `json:"isMbUser"`
	IsSnsUser int    `json:"isSnsUser"`
	Mobile    string `json:"mobile"`
}

type UserDataV2 struct {
	Uid       string `json:"uid"`
	Username  string `json:"username"`
	Mobile    string `json:"mobile"`
	IsGuest   string `json:"isGuest"`
	RegDevice string `json:"regDevice"`
	SexType   string `json:"sexType"`
	IsMbUser  int    `json:"isMbUser"`
	IsSnsUser int    `json:"isSnsUser"`
	Token     string `json:"token"`
}

type PtConfig struct {
	UseSms            string       `json:"useSms"`
	FcmTips           *FcmTips     `json:"fcmTips"`
	JoinQQGroup       *JoinQQGroup `json:"joinQQGroup"`
	ServiceInfo       string       `json:"serviceInfo"`
	IsFloat           string       `json:"isFloat"`
	MainLogin         string       `json:"mainLogin"`
	UseService        string       `json:"useService"`
	UseServiceCenter  string       `json:"useServiceCenter"`
	Logo              string       `json:"logo"`
	UseBBS            string       `json:"useBBS"`
	Gift              string       `json:"gift"`
	IsShowFloat       string       `json:"isShowFloat"`
	AutoOpenAgreement string       `json:"autoOpenAgreement"`
	MainLoginType     string       `json:"mainLoginType"`
	UcentUrl          string       `json:"ucentUrl"`
	UseCpLogin        string       `json:"useCpLogin"`
	FloatLogo         string       `json:"floatLogo"`
	Theme             string       `json:"theme"`
	UseAppAuth        string       `json:"useAppAuth"`
	SwitchWxAppPlug   string       `json:"switchWxAppPlug"`
	IdverifyTipTit    string       `json:"idverifyTipTit"`
	BanshuSwitch      string       `json:"banshuSwitch"`
	RmAccountLg       string       `json:"rmAccountLg"`
	RegVerifyCode     string       `json:"regVerifyCode"`
	DisFastReg        string       `json:"disFastReg"`
	NoPassWallet      string       `json:"noPassWallet"`
	HideMyFunc        *HideMyFunc  `json:"hideMyFunc"`
	Title             string       `json:"title"`
	SkinStyle         string       `json:"skinStyle"`
	RmGuestLg         int          `json:"rmGuestLg"`
}

type HideMyFunc struct {
	HideRegBtn          int `json:"hideRegBtn"`
	CustAdReport        int `json:"custAdReport"`
	NormalUserBindPhone int `json:"normalUserBindPhone"`
	EnableEvt           int `json:"enableEvt"`
}

type FcmTips struct {
	NoAdultLogoutTip string `json:"noAdultLogoutTip"`
	GuestLoginTip    string `json:"guestLoginTip"`
	MinorLoginTip    string `json:"minorLoginTip"`
	MinorTimeTip     string `json:"minorTimeTip"`
	AgeLimitTip      string `json:"ageLimitTip"`
	AgeMaxLimitTip   string `json:"ageMaxLimitTip"`
	NoAdultCommonTip string `json:"noAdultCommonTip"`
	ShiMingTip8      string `json:"shiMingTip8"`
	ShiMingTip816    string `json:"shiMingTip8_16"`
	ShiMingTip1618   string `json:"shiMingTip16_18"`
	GuestTimeTip     string `json:"guestTimeTip"`
	HolidayTimeTip   string `json:"holidayTimeTip"`
	Less8PayTip      string `json:"less8PayTip"`
}

type JoinQQGroup struct {
	GroupNum string `json:"groupNum"`
	GroupKey string `json:"groupKey"`
}

type PtVer struct {
	VersionName string `json:"versionName"`
	VersionNo   int    `json:"versionNo"`
	VersionUrl  string `json:"versionUrl"`
	UpdateTime  string `json:"updateTime"`
	IsMust      string `json:"isMust"`
	UpdateTips  string `json:"updateTips"`
}

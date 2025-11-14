package ussd

// USSDRequest represents an incoming USSD request
// This follows the Africa's Talking USSD API format
type USSDRequest struct {
	SessionID   string `json:"sessionId" form:"sessionId"`
	ServiceCode string `json:"serviceCode" form:"serviceCode"`
	PhoneNumber string `json:"phoneNumber" form:"phoneNumber"`
	Text        string `json:"text" form:"text"`
	NetworkCode string `json:"networkCode" form:"networkCode,omitempty"`
}

// USSDResponse represents a USSD response
type USSDResponse struct {
	Message string
	Type    ResponseType
}

// ResponseType indicates if the session should continue or end
type ResponseType string

const (
	// Continue indicates more input is expected (CON)
	Continue ResponseType = "CON"

	// End indicates the session should terminate (END)
	End ResponseType = "END"
)

// String returns the formatted USSD response
func (r USSDResponse) String() string {
	return string(r.Type) + " " + r.Message
}

// MenuOption represents a single menu option
type MenuOption struct {
	Key         string
	Label       string
	Description string
}

// MenuPage represents a paginated menu
type MenuPage struct {
	Items      []MenuItem
	PageNumber int
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

// MenuItem represents a single item in a menu
type MenuItem struct {
	ID          string
	Label       string
	Description string
	Value       interface{}
}

// SessionData represents the data stored in a USSD session
type SessionData struct {
	CurrentMenu string                 `json:"current_menu"`
	MenuStack   []string               `json:"menu_stack"`
	PageNumber  int                    `json:"page_number"`
	Data        map[string]interface{} `json:"data"`
	CustomerID  string                 `json:"customer_id,omitempty"`
	TenantID    string                 `json:"tenant_id,omitempty"`
}

// NewSessionData creates a new session data instance
func NewSessionData() *SessionData {
	return &SessionData{
		CurrentMenu: "main",
		MenuStack:   []string{},
		PageNumber:  1,
		Data:        make(map[string]interface{}),
	}
}

// PushMenu adds a menu to the stack
func (sd *SessionData) PushMenu(menu string) {
	sd.MenuStack = append(sd.MenuStack, sd.CurrentMenu)
	sd.CurrentMenu = menu
	sd.PageNumber = 1
}

// PopMenu removes the last menu from the stack
func (sd *SessionData) PopMenu() string {
	if len(sd.MenuStack) == 0 {
		return "main"
	}

	lastIdx := len(sd.MenuStack) - 1
	previous := sd.MenuStack[lastIdx]
	sd.MenuStack = sd.MenuStack[:lastIdx]
	sd.CurrentMenu = previous
	sd.PageNumber = 1
	return previous
}

// SetData stores a value in the session data
func (sd *SessionData) SetData(key string, value interface{}) {
	sd.Data[key] = value
}

// GetData retrieves a value from the session data
func (sd *SessionData) GetData(key string) (interface{}, bool) {
	value, ok := sd.Data[key]
	return value, ok
}

// GetDataString retrieves a string value from the session data
func (sd *SessionData) GetDataString(key string) (string, bool) {
	value, ok := sd.GetData(key)
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// Menu represents a USSD menu interface
type Menu interface {
	Render(session *SessionData) USSDResponse
	Handle(input string, session *SessionData) (nextMenu string, response USSDResponse)
}

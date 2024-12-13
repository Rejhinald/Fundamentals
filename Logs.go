package models

import (
	"grooper/app/utils"

	"github.com/revel/revel"
)

type LogModuleParams struct {
	ID            string            `json:"ID,omitempty"`
	Status        string            `json:"Status,omitempty"`
	CreatedAt     string            `json:"CreatedAt,omitempty"`
	Email         string            `json:"Email,omitempty"`
	Name          string            `json:"Name,omitempty"`
	CategoryName  string            `json:"CategoryName,omitempty"`
	Categories    []string          `json:"Categories,omitempty"`
	Content       string            `json:"Content,omitempty"`
	DisplayPhoto  string            `json:"DisplayPhoto,omitempty"`
	Bg            string            `json:"Bg,omitempty"`
	Type          string            `json:"Type,omitempty"`
	GroupName     string            `json:"GroupName,omitempty"`
	Old           interface{}       `json:"Old,omitempty"`
	New           interface{}       `json:"New,omitempty"`
	Origin        interface{}       `json:"Origin,omitempty"`
	PersonalEmail string            `json:"PersonalEmail,omitempty"`
	GoogleEmail   string            `json:"GoogleEmail,omitempty"`
	Temp          TempParam         `json:"Temp,omitempty"`
	Members       interface{}       `json:"Members,omitempty"`
	Group         *LogModuleParams  `json:"Group,omitempty"`
	Project       *LogModuleParams  `json:"Project,omitempty"`
	Roles         *IntegrationRoles `json:"Roles,omitempty"`
}

type TempParam struct {
	ID     string
	Name   string
	Status string
}

type IntegrationRoles struct {
	New interface{}
	Old interface{}
}

type LogInformation struct {
	Action               string            `json:"Action,omitempty"`
	PerformedBy          string            `json:"PerformedBy,omitempty"` // To be removed
	Author               LogModuleParams   `json:"Author,omitempty"`
	User                 *LogModuleParams  `json:"User,omitempty"`
	Users                []LogModuleParams `json:"Users,omitempty"`
	Groups               []LogModuleParams `json:"Groups,omitempty"`
	Group                *LogModuleParams  `json:"Group,omitempty"`
	Members              []LogModuleParams `json:"Members,omitempty"`
	SourceGroup          *LogModuleParams  `json:"source_group,omitempty"`
	DestinationGroup     *LogModuleParams  `json:"destination_group,omitempty"`
	RenamedGroup         *LogModuleParams  `json:"RenamedGroup,omitempty"`
	RetainedGroup        *LogModuleParams  `json:"RetainedGroup,omitempty"`
	Origin               *LogModuleParams  `json:"Origin,omitempty"`
	RemovedGroup         *LogModuleParams  `json:"RemovedGroup,omitempty"`
	Department           *LogModuleParams  `json:"Department,omitempty"`
	GroupMember          *LogModuleParams  `json:"GroupMember,omitempty"`
	GroupMembers         []LogModuleParams `json:"GroupMembers,omitempty"`
	Subscription         *LogModuleParams  `json:"Subscription,omitempty"`
	Company              *LogModuleParams  `json:"Company,omitempty"`
	Integration          *LogModuleParams  `json:"Integration,omitempty"`
	SubIntegration       *LogModuleParams  `json:"SubIntegration,omitempty"`
	SuggestedIntegration *LogModuleParams  `json:"SuggestedIntegration,omitempty"`
	Integrations         []LogModuleParams `json:"Integrations,omitempty"`
	Role                 *LogModuleParams  `json:"Role,omitempty"`
	Roles                []LogModuleParams `json:"Roles,omitempty"`
	Permissions          []LogModuleParams `json:"Permissions,omitempty"`
	Category             *LogModuleParams  `json:"Category,omitempty"`
	Feedback             *LogModuleParams  `json:"Feedback,omitempty"`
	InvitedUser          *LogModuleParams  `json:"InvitedUserAttributes,omitempty"`
}

type Logs struct {
	PK        string          `json:"PK,omitempty"`
	SK        string          `json:"SK,omitempty"`
	LogID     string          `json:"LogID,omitempty"`
	CompanyID string          `json:"CompanyID,omitempty"`
	GroupID   string          `json:"GroupID,omitempty"`
	UserID    string          `json:"UserID,omitempty"` // PerformedBy
	LogType   string          `json:"LogType,omitempty"`
	LogAction string          `json:"LogAction,omitempty"`
	CreatedAt string          `json:"CreatedAt,omitempty"`
	LogInfo   *LogInformation `json:"LogInfo,omitempty"`
	Type      string          `json:"Type,omitempty"`
	SearchKey string          `json:"SearchKey,omitempty"`
}

func (model *Logs) Validate(v *revel.Validation) {
	utils.ValidateID(v, model.LogID).Key("logID")
	utils.ValidateRequired(v, model.LogType).Key("logType")
	utils.ValidateRequired(v, model.LogInfo).Key("logInfo")
}

type NewLogs struct {
	PK        string          `json:"PK,omitempty"`
	SK        string          `json:"SK,omitempty"`
	LogID     string          `json:"LogID,omitempty"`
	CompanyID string          `json:"CompanyID,omiempty"`
	UserID    string          `json:"UserID,omitempty"`
	LogInfo   *LogInformation `json:"LogInfo,omitempty"`
	LogEvent  string          `json:"LogEvent,omitempty"`
	CreatedAt string          `json:"CreatedAt,omitempty"`
	SourceIP  string          `json:"SourceIP,omitempty"`
	Level     string          `json:"Level,omitempty"`
}

func (model *NewLogs) Validate(v *revel.Validation) {
	utils.ValidateRequired(v, model.LogEvent).Key("logEvent")
	utils.ValidateRequired(v, model.LogInfo).Key("logInfo")
}

type InvitedUserAttributes struct {
}

// @Description The success response for the logs
// TODO this is the model responses to use for GetLogs
type GetLogsResponse struct {
	Status           string `json:"status"`
	Logs             []Logs `json:"logs"`
	LastEvaluatedKey *Logs  `json:"lastEvaluatedKey,omitempty"`
}

type GetLogSuccessResponse struct {
	Status string `json:"status"`
	Logs   []Logs `json:"logs"`
}
type GetNewLogsSuccessResponse struct {
	Status           string    `json:"status"`
	NewLogs          []NewLogs `json:"newlogs"`
	LastEvaluatedKey *Logs     `json:"lastEvaluatedKey,omitempty"`
}

type DeleteLogsSuccessResponse struct {
	Status interface{} `json:"status"`
}

package constants

type Status struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Standard HTTP Status Codes
const HTTP_STATUS_200 = "200"
const HTTP_STATUS_201 = "201"
const HTTP_STATUS_304 = "304"
const HTTP_STATUS_400 = "400"
const HTTP_STATUS_401 = "401"
const HTTP_STATUS_403 = "403"
const HTTP_STATUS_404 = "404"
const HTTP_STATUS_406 = "406"
const HTTP_STATUS_405 = "405"
const HTTP_STATUS_409 = "409"
const HTTP_STATUS_410 = "410"
const HTTP_STATUS_411 = "411"
const HTTP_STATUS_412 = "412"
const HTTP_STATUS_413 = "413"
const HTTP_STATUS_415 = "415"
const HTTP_STATUS_416 = "416"
const HTTP_STATUS_422 = "422"
const HTTP_STATUS_423 = "423"
const HTTP_STATUS_429 = "429"
const HTTP_STATUS_500 = "500"
const HTTP_STATUS_501 = "501"
const HTTP_STATUS_502 = "502"
const HTTP_STATUS_503 = "503"
const HTTP_STATUS_504 = "504"
const HTTP_STATUS_507 = "507"
const HTTP_STATUS_509 = "509"

// Custom HTTP Status Codes
const HTTP_STATUS_460 = "460"
const HTTP_STATUS_461 = "461"
const HTTP_STATUS_462 = "462"
const HTTP_STATUS_463 = "463"
const HTTP_STATUS_464 = "464"
const HTTP_STATUS_465 = "465"
const HTTP_STATUS_466 = "466"
const HTTP_STATUS_467 = "467"
const HTTP_STATUS_468 = "468"
const HTTP_STATUS_469 = "469"
const HTTP_STATUS_470 = "470"
const HTTP_STATUS_471 = "471"
const HTTP_STATUS_472 = "472"
const HTTP_STATUS_473 = "473"
const HTTP_STATUS_474 = "474"
const HTTP_STATUS_475 = "475"
const HTTP_STATUS_476 = "476"
const HTTP_STATUS_477 = "477"
const HTTP_STATUS_478 = "478"
const HTTP_STATUS_479 = "479"
const HTTP_STATUS_480 = "480"
const HTTP_STATUS_481 = "481"
const HTTP_STATUS_482 = "482"
const HTTP_STATUS_483 = "483"
const HTTP_STATUS_484 = "484"
const HTTP_STATUS_485 = "485"
const HTTP_STATUS_486 = "486"
const HTTP_STATUS_487 = "487"
const HTTP_STATUS_488 = "488"
const HTTP_STATUS_489 = "489"
const HTTP_STATUS_490 = "490"
const HTTP_STATUS_495 = "495"
const HTTP_STATUS_496 = "496"
const HTTP_STATUS_497 = "497"

// Integration Already exist
const HTTP_STATUS_492 = "492"
const HTTP_STATUS_493 = "493"

// Stripe
const HTTP_STATUS_494 = "494"

// GitHub
const HTTP_STATUS_700 = "700"

// Slack
const HTTP_STATUS_701 = "701"
const HTTP_STATUS_702 = "702"
const HTTP_STATUS_703 = "703"
const HTTP_STATUS_704 = "704"
const HTTP_STATUS_705 = "705"
const HTTP_STATUS_706 = "706"
const HTTP_STATUS_707 = "707"

// UserGroup
const HTTP_STATUS_801 = "801"
const HTTP_STATUS_802 = "802"
const HTTP_STATUS_803 = "803"
const HTTP_STATUS_804 = "804"
const HTTP_STATUS_805 = "805"
const HTTP_STATUS_806 = "806"

// Hootsuite
const HTTP_STATUS_3006 = "3006"
const HTTP_STATUS_3010 = "3010"

// Asana
const HTTP_STATUS_901 = "901"
const HTTP_STATUS_902 = "902"
const HTTP_STATUS_903 = "903"

var HTTP_STATUS = map[string]Status{
	"200": {
		Code:    "200",
		Message: "OK. The request has succeeded.",
	},
	"201": {
		Code:    "201",
		Message: "CREATED. The request was successfully fulfilled and resulted in one or possibly multiple new resources being created",
	},
	"304": {
		Code:    "304",
		Message: "NOT MODIFIED. The website you're requesting hasn't been updated since the last time you accessed it.",
	},
	"400": {
		Code:    "400",
		Message: "BAD REQUEST. The server could not understand the request due to invalid syntax.",
	},
	"401": {
		Code:    "401",
		Message: "UNAUTHORIZED. The authentication credentials are missing, or if supplied are not valid or not sufficient to access the resource.",
	},
	"403": {
		Code:    "403",
		Message: "FORBIDDEN: The client does not have access rights to the content; that is, it is unauthorized, so the server is refusing to give the requested resource.",
	},
	"404": {
		Code:    "404",
		Message: "NOT FOUND: The URI requested is invalid or the resource requested does not exists.",
	},
	"405": {
		Code:    "405",
		Message: "METHOD NOT ALLOWED: Request method is known by the server but is not supported by the target resource.",
	},
	"406": {
		Code:    "406",
		Message: "NOT ACCEPTABLE: Your website or web application does not support the client's request with a particular protocol.",
	},
	"409": {
		Code:    "409",
		Message: "CONFLICT: The request conflicts with the current state of the server.",
	},
	"410": {
		Code:    "410",
		Message: "GONE: This resource is gone. Used to indicate that an API endpoint has been turned off.",
	},
	"411": {
		Code:    "411",
		Message: "LENGTH REQUIRED: The server refuses to accept the request without a defined Content-Length header.",
	},
	"412": {
		Code:    "412",
		Message: "PRECONDITION FAILED: The access to the target resource has been denied",
	},
	"413": {
		Code:    "413",
		Message: "REQUEST ENTITY TOO LARGE: The request is larger than the server is willing or able to process.",
	},
	"415": {
		Code:    "415",
		Message: "UNSUPPORTED MEDIA TYPE: The server does not support media type",
	},
	"416": {
		Code:    "416",
		Message: "REQUESTED RANGE NOT SATISFIABLE: The client has asked for unprovidable portion of the file",
	},
	"422": {
		Code:    "422",
		Message: "UNPROCESSABLE ENTITY: The request was well-formed but was unable to be followed due to semantic errors.",
	},
	"423": {
		Code:    "423",
		Message: "LOCKED: The resource that is being accessed is locked.",
	},
	"429": {
		Code:    "429",
		Message: "TOO MANY REQUESTS: Returned when a request cannot be served due to the application’s rate limit having been exhausted for the resource.",
	},
	"460": {
		Code:    "460",
		Message: "Company ID does not exist.",
	},
	"461": {
		Code:    "461",
		Message: "User ID does not exist.",
	},
	"462": {
		Code:    "462",
		Message: "Email does not exist",
	},
	"463": {
		Code:    "463",
		Message: "Department already exists in this company.",
	},
	"464": {
		Code:    "464",
		Message: "User already exists in this company",
	},
	"465": {
		Code:    "465",
		Message: "Invalid role.",
	},
	"466": {
		Code:    "466",
		Message: "Invalid status",
	},
	"467": {
		Code:    "467",
		Message: "Authorization header not found.",
	},
	"468": {
		Code:    "468",
		Message: "Token format is invalid",
	},
	"469": {
		Code:    "469",
		Message: "Group already exists in this department.",
	},
	"470": {
		Code:    "470",
		Message: "Department ID does not exists.",
	},
	"471": {
		Code:    "471",
		Message: "Unknown entity type.",
	},
	"472": {
		Code:    "472",
		Message: "Password must contain at least 8 characters, one uppercase and lowercase letter and one number",
	},
	"473": {
		Code:    "473",
		Message: "Email already exists.",
	},
	"474": {
		Code:    "474",
		Message: "Group ID does not exists.",
	},
	"475": {
		Code:    "475",
		Message: "Password reset token is invalid.",
	},
	"476": {
		Code:    "476",
		Message: "An error has occurred while uploading the file to AWS S3",
	},
	"477": {
		Code:    "477",
		Message: "Company already exists",
	},
	"478": {
		Code:    "478",
		Message: "Password is incorrect.",
	},
	"479": {
		Code:    "479",
		Message: "Unique Identifier not found",
	},
	"480": {
		Code:    "480",
		Message: "Creating user a microsoft account failed.",
	},
	"481": {
		Code:    "481",
		Message: "Inviting user failed.",
	},
	"482": {
		Code:    "482",
		Message: "Removing user failed.",
	},
	"483": {
		Code:    "483",
		Message: "Removing user to azure group failed.",
	},
	"484": {
		Code:    "484",
		Message: "Adding user failed.",
	},
	"485": {
		Code:    "485",
		Message: "Not enough license seat.",
	},
	"486": {
		Code:    "486",
		Message: "Assigning license to user failed.",
	},
	"487": {
		Code:    "487",
		Message: "Removing license to user failed.",
	},
	"488": {
		Code:    "488",
		Message: "Unknown Error",
	},
	"489": {
		Code:    "489",
		Message: "Invalid Verification Link",
	},
	"490": {
		Code:    "490",
		Message: "Group name already exists.",
	},
	"491": {
		Code:    "491",
		Message: "Must be a Google Workspace account.",
	},
	"492": {
		Code:    "492",
		Message: "Integration name already exists.",
	},
	"493": {
		Code:    "493",
		Message: "Integration slug already exists.",
	},
	"494": {
		Code:    "494",
		Message: "Stripe card already exists",
	},
	"495": {
		Code:    "495",
		Message: "The new password cannot be the same as the current password.",
	},
	"496": {
		Code:    "496",
		Message: "Subscription not found.",
	},
	"497": {
		Code:    "497",
		Message: "Request already submitted.",
	},
	"500": {
		Code:    "500",
		Message: "INTERNAL SERVER ERROR. The server encountered an unexpected condition which prevented it from fulfilling the request.",
	},
	"501": {
		Code:    "501",
		Message: "NOT IMPLEMENTED: The server does not support the facility required.",
	},
	"502": {
		Code:    "502",
		Message: "Bad Gateway. The server was acting as a gateway or proxy and received an invalid response from the upstream server.",
	},
	"503": {
		Code:    "503",
		Message: "SERVICE UNAVAILABLE: the server is not ready to handle the request.",
	},
	"504": {
		Code:    "504",
		Message: "GATEWAY TIMEOUT: The gateway did not receive response from upstream server.",
	},
	"507": {
		Code:    "507",
		Message: "INSUFFICIENT STORAGE: The server is unable to store the representation.",
	},
	"509": {
		Code:    "509",
		Message: "BANDWIDTH LIMIT EXCEEDED: The bandwidth limit exceeded.",
	},
	"700": {
		Code:    "700",
		Message: "GitHub username not found.",
	},
	"701": {
		Code:    "701",
		Message: "Channel name already exist.",
	},
	"702": {
		Code:    "702",
		Message: "Value passed contains unallowed special characters or uppercase characters.",
	},
	"703": {
		Code:    "703",
		Message: "Only workspace admin/owner can remove members from the channel.",
	},
	"704": {
		Code:    "704",
		Message: "Authenticated user can't remove itself from a channel.",
	},
	"705": {
		Code:    "705",
		Message: "User cannot be removed from #general.",
	},
	"706": {
		Code:    "706",
		Message: "This feature is only available to paid teams.",
	},
	"707": {
		Code:    "707",
		Message: "Only the user that originally created the channel or an admin may rename it.",
	},
	"801": {
		Code:    "801",
		Message: "User has no permission to create a User group.",
	},
	"802": {
		Code:    "802",
		Message: "Usergroup name already exists;.",
	},
	"803": {
		Code:    "803",
		Message: "Usergroup handle already exists;.",
	},
	"804": {
		Code:    "804",
		Message: "Usergroup handle is invalid;.",
	},
	"805": {
		Code:    "805",
		Message: "Usergroups can only be used on paid Slack teams",
	},
	"806": {
		Code:    "806",
		Message: "Usergroups can only be accessed by Standard plans or above",
	},
	"3006": {
		Code:    "3006",
		Message: "Team name already exists.",
	},
	"3010": {
		Code:    "3010",
		Message: "Email already in use.",
	},
	"901": {
		Code:    "901",
		Message: "User is not recognized.",
	},
	"902": {
		Code:    "902",
		Message: "User is not in organization.",
	},
	"903": {
		Code:    "903",
		Message: "Project name already exists.",
	},
}

//Table Attributes
// const TABLE_NAME = "grooper"

const TABLE_NAME = "db.table"
const LOGS_TABLE_NAME = "db.logsTable"

const DEFAULT_PAGE_LIMIT = 50

// Comparison Operators
var CONDITION = map[string]string{
	"equal":                 "EQ",
	"not_equal":             "NE",
	"less_than_or_equal":    "LE",
	"less_than":             "LT",
	"greater_than_or_equal": "GE",
	"greater_than":          "GT",
	"not_null":              "NOT_NULL",
	"null":                  "NULL",
	"contains":              "CONTAINS",
	"not_contains":          "NOT_CONTAINS",
	"begins_with":           "BEGINS_WITH",
	"in":                    "IN",
	"between":               "BETWEEN",
}

// Comparison Operators
const CONDITION_EQUAL = "EQ"
const CONDITION_NOT_EQUAL = "NE"
const CONDITION_LESS_THAN_OR_EQUAL = "LE"
const CONDITION_LESS_THAN = "LT"
const CONDITION_GREATER_THAN_OR_EQUAL = "GE"
const CONDITION_GREATER_THAN = "GT"
const CONDITION_NOT_NULL = "NOT NULL"
const CONDITION_NULL = "NULL"
const CONDITION_CONTAINS = "CONTAINS"
const CONDITION_NOT_CONTAINS = "NOT_CONTAINS"
const CONDITION_BEGINS_WITH = "BEGINS_WITH"
const CONDITION_IN = "IN"
const CONDITION_BETWEEN = "BETWEEN"

// Prefix
var PREFIX = map[string]string{
	"user":               "USER#",
	"company":            "COMPANY#",
	"department":         "DEPARTMENT#",
	"group":              "GROUP#",
	"subscription":       "SUBSCRIPTION#",
	"logs":               "LOG#",
	"role":               "ROLE#",
	"permission":         "PERMISSION#",
	"owner":              "OWNER#",
	"on_boarding_member": "ONBOARDINGMEMBER#",
}

const PREFIX_USER = "USER#"
const PREFIX_COMPANY = "COMPANY#"
const PREFIX_DEPARTMENT = "DEPARTMENT#"
const PREFIX_GROUP = "GROUP#"
const PREFIX_OWNER = "OWNER#"
const PREFIX_SUBSCRIPTION = "SUBSCRIPTION#"
const PREFIX_LOG = "LOG#"
const PREFIX_ROLE = "ROLE#"
const PREFIX_PERMISSION = "PERMISSION#"
const PREFIX_INTEGRATION = "INTEGRATION#"
const PREFIX_SUB_INTEGRATION = "SUB_INTEGRATION#"
const PREFIX_CATEGORY = "CATEGORY#"
const PREFIX_INTEGRATION_REQUEST = "INTEGRATION_REQUEST#"
const PREFIX_NOTIFICATION = "NOTIFICATION#"
const PREFIX_ACTION_ITEM = "ACTION_ITEM#"
const PREFIX_SETTINGS = "SETTINGS#"
const PREFIX_FEEDBACK = "FEEDBACK#"
const PREFIX_DATE = "DATE#"
const PREFIX_CORRELATION_ID = "CORRELATIONID#"
const PREFIX_JOB = "JOB#"
const PREFIX_OAUTH = "OAUTH#"
const PREFIX_WHATS_NEW = "WHATS_NEW#"
const PREFIX_PERMISSION_GROUP = "PERMISSION_GROUP#"


// EntityType
var ENTITY_TYPE = map[string]string{
	"user":              "USER",
	"company":           "COMPANY",
	"department":        "DEPARTMENT",
	"group":             "GROUP",
	"group_member":      "GROUPMEMBER",
	"group_owner":       "GROUPOWNER",
	"company_member":    "COMPANYMEMBER",
	"department_member": "DEPARTMENTMEMBER",
	"subscription":      "SUBSCRIPTION",
	"log":               "LOG",
	"integration":       "INTEGRATION",
	"PERMISSION":        "PERMISSION",
	"whats_new":         "WHATS_NEW",
}

const ENTITY_TYPE_AUTH = "AUTH"
const ENTITY_TYPE_USER = "USER"
const ENTITY_TYPE_COMPANY = "COMPANY"
const ENTITY_TYPE_DEPARTMENT = "DEPARTMENT"
const ENTITY_TYPE_GROUP = "GROUP"
const ENTITY_TYPE_GROUP_MEMBER = "GROUPMEMBER"
const ENTITY_TYPE_GROUP_OWNER = "GROUPOWNER"
const ENTITY_TYPE_GROUP_INTEGRATION = "GROUPINTEGRATION"
const ENTITY_TYPE_COMPANY_INTEGRATION = "COMPANYINTEGRATION"
const ENTITY_TYPE_GROUP_SUB_INTEGRATION = "GROUPSUBINTEGRATION"
const ENTITY_TYPE_COMPANY_SUB_INTEGRATION = "COMPANYSUBINTEGRATION"
const ENTITY_TYPE_COMPANY_MEMBER = "COMPANYMEMBER"
const ENTITY_TYPE_DEPARTMENT_MEMBER = "DEPARTMENTMEMBER"
const ENTITY_TYPE_SUBSCRIPTION = "SUBSCRIPTION"
const ENTITY_TYPE_SUBSCRIPTION_USER = "SUBSCRIPTIONUSER"
const ENTITY_TYPE_LOG = "LOG"
const ENTITY_TYPE_INTEGRATION = "INTEGRATION"
const ENTITY_TYPE_SUB_INTEGRATION = "SUB_INTEGRATION"
const ENTITY_TYPE_ROLE = "ROLE"
const ENTITY_TYPE_USER_ROLE = "USERROLE"
const ENTITY_TYPE_PERMISSION = "PERMISSION"
const ENTITY_TYPE_USER_INTEGRATION_UID = "USERINTEGRATIONUID"
const ENTITY_TYPE_FEEDBACK = "FEEDBACK"
const ENTITY_TYPE_ORACLE_DB_OAUTH = "ORACLE_DB_OAUTH"
const ENTITY_TYPE_WHATS_NEW = "WHATS_NEW"
const ENTITY_TYPE_PERMISSION_GROUPS = "PERMISSION_GROUPS"

// IndexName
var INDEX_NAME = map[string]string{
	"inverted_index":    "InvertedIndex",
	"get_users_by_role": "GetUsersByRole",
	"get_users":         "GetUsers",
	"get_companies":     "GetCompanies",
	"get_departments":   "GetDepartments",
	"get_groups":        "GetGroups",
	"get_integrations":  "GetIntegrations",
	"get_logs":          "GetLogs",
}

const INDEX_NAME_INVERTED_INDEX = "InvertedIndex"
const INDEX_NAME_GET_USERS_BY_ROLE = "GetUsersByRole"
const INDEX_NAME_GET_USERS = "GetUsers"
const INDEX_NAME_GET_COMPANIES = "GetCompanies"
const INDEX_NAME_GET_DEPARTMENTS = "GetDepartments"
const INDEX_NAME_GET_GROUPS = "GetGroups"
const INDEX_NAME_GET_INTEGRATIONS = "GetIntegrations"
const INDEX_NAME_GET_LOGS = "GetLogs"
const INDEX_NAME_GET_ROLES = "GetRoles"
const INDEX_NAME_GET_PERMISSION = "GetPermissions"
const INDEX_NAME_GET_SUBSCRIPTION = "GetSubscriptions"
const INDEX_NAME_GET_SUBSCRIPTION_NEW = "GetSubscriptionsNew"
const INDEX_NAME_GET_COMPANY_ITEMS = "GetCompanyItems"
const INDEX_NAME_GET_COMPANY_ITEMS_NEW = "GetCompanyItemsNew"
const INDEX_NAME_GET_COMPANY_ITEMS_BY_SK = "GetCompanyItemsBySK"
const INDEX_NAME_GET_COMPANY_USERS_BY_HANDLER = "GetCompanyUsersByHandler"

// API Actions
var ACTION = map[string]string{
	"update": "UPDATE",
	"delete": "DELETE",
	"add":    "ADD",
	"move":   "MOVE",
	"merge":  "MERGE",
	"clone":  "CLONE",
}

// API Actions
const ACTION_ADD = "ADD"
const ACTION_UPDATE = "UPDATE"
const ACTION_DELETE = "DELETE"
const ACTION_MOVE = "MOVE"
const ACTION_MERGE = "MERGE"
const ACTION_CLONE = "CLONE"
const ACTION_BRANCH = "BRANCH"
const ACTION_REQUEST = "REQUEST"

// PERMISSION CATEGORY
const PERMISSION_CATEGORY_GROUP_PERMISSION = "GROUP_PERMISSION"
const PERMISSION_CATEGORY_COMPANY_PERMISSION = "COMPANY_PERMISSION"
const PERMISSION_CATEGORY_DEPARTMENT_PERMISSION = "DEPARTMENT_PERMISSION"

// Service Type - Mainly used for form validation in models
var SERVICE_TYPE = map[string]string{
	//Users
	"signup_user":        "SIGNUP_USER",
	"get_user":           "GET_USER",
	"add_user":           "ADD_USER",
	"update_user":        "UPDATE_USER",
	"update_user_status": "UPDATE_USER_STATUS",
	"update_user_email":  "UPDATE_USER_EMAIL",
}

// Service Type - Mainly used for form validation in models
const SERVICE_TYPE_GET_USER = "GET_USER"
const SERVICE_TYPE_ADD_USER = "ADD_USER"
const SERVICE_TYPE_UPDATE_USER = "UPDATE_USER"
const SERVICE_TYPE_UPDATE_USER_PROFILE = "UPDATE_USER_PROFILE"
const SERVICE_TYPE_UPDATE_USER_STATUS = "UPDATE_USER_STATUS"
const SERVICE_TYPE_UPDATE_USER_EMAIL = "UPDATE_USER_EMAIL"
const SERVICE_TYPE_ADD_COMPANY = "ADD_COMPANY"
const SERVICE_TYPE_SIGNUP_USER = "SIGNUP_USER"
const SERVICE_TYPE_CREATE_ROLE = "CREATE_ROLE"
const SERVICE_TYPE_ASSIGN_ROLE = "ASSIGN_ROLE"
const SERVICE_TYPE_UPDATE_ROLE = "UPDATE_ROLE"
const SERVICE_TYPE_CREATE_PERMISSION = "CREATE_PERMISSION"
const SERVICE_TYPE_ASSIGN_PERMISSION = "ASSIGN_PERMISSION"
const SERVICE_TYPE_UPDATE_PERMISSION = "UPDATE_PERMISSION"
const SERVICE_TYPE_UPDATE_ACTIVE_COMPANY = "SERVICE_TYPE_UPDATE_ACTIVE_COMPANY"
const SERVICE_TYPE_CREATE_SUBSCRIPTION = "SERVICE_TYPE_CREATE_SUBSCRIPTION"
const SERVICE_TYPE_MANAGE_ACCESS = "USER_MANAGE_ACCESS"

// Status
var ITEM_STATUS = map[string]string{
	"active":   "ACTIVE",
	"deleted":  "DELETED",
	"inactive": "INACTIVE",
}

// Item Status
const ITEM_STATUS_ACTIVE = "ACTIVE"
const ITEM_STATUS_DELETED = "DELETED"
const ITEM_STATUS_INACTIVE = "INACTIVE"
const ITEM_STATUS_REVOKED = "REVOKED"
const ITEM_STATUS_PENDING = "PENDING"
const ITEM_STATUS_EXPIRED = "EXPIRED"
const ITEM_STATUS_CANCEL = "CANCEL"
const ITEM_STATUS_PERMANENTLY_DELETED = "PERMANENTLY_DELETED"
const ITEM_STATUS_DEFAULT = "DEFAULT"
const ITEM_STATUS_CANCELED = "CANCELED"
const ITEM_STATUS_NOT_INVITED = "NOT_INVITED"
const ITEM_STATUS_DONE = "DONE"
const ITEM_STATUS_SCHEDULED = "SCHEDULED"

// User Role
var USER_ROLE = map[string]string{
	"user":   "USER",
	"admin":  "ADMIN",
	"viewer": "VIEWER",
}

// User Role
const USER_ROLE_USER = "USER"
const USER_ROLE_ADMIN = "ADMIN"
const USER_ROLE_VIEWER = "VIEWER"
const USER_ROLE_COMPANY_OWNER = "COMPANY_OWNER"

// Member Type
const MEMBER_TYPE_USER = "USER"
const MEMBER_TYPE_OWNER = "OWNER"
const MEMBER_TYPE_GROUP = "GROUP"

var TRIM_TYPE = map[string]string{
	"all":    "ALL",
	"left":   "LEFT",
	"right":  "RIGHT",
	"prefix": "PREFIX",
	"suffix": "SUFFIX",
}

// Trim Type
const TRIM_TYPE_ALL = "ALL"
const TRIM_TYPE_LEFT = "LEFT"
const TRIM_TYPE_RIGHT = "RIGHT"
const TRIM_TYPE_PREFIX = "PREFIX"
const TRIM_TYPE_SUFFIX = "SUFFIX"

const BATCH_LIMIT = 25

// User Permissions
const USER_PERMISSION_ADD_PEOPLE = "ADD_PEOPLE"
const USER_PERMISSION_DELETE_PEOPLE = "DELETE_PEOPLE"
const USER_PERMISSION_EDIT_PEOPLE = "EDIT_PEOPLE"
const USER_PERMISSION_ADD_GROUP = "ADD_GROUP"
const USER_PERMISSION_EDIT_GROUP = "EDIT_GROUP"
const USER_PERMISSION_DELETE_GROUP = "DELETE_GROUP"
const USER_PERMISSION_CLONE_GROUP = "CLONE_GROUP"
const USER_PERMISSION_BRANCH_GROUP = "BRANCH_GROUP"
const USER_PERMISSION_MERGE_GROUP = "MERGE_GROUP"
const USER_PERMISSION_ADD_COMPANY = "ADD_COMPANY"
const USER_PERMISSION_EDIT_COMPANY = "EDIT_COMPANY"
const USER_PERMISSION_DELETE_COMPANY = "DELETE_COMPANY"
const USER_PERMISSION_ADD_DEPARTMENT = "ADD_DEPARTMENT"
const USER_PERMISSION_EDIT_DEPARTMENT = "EDIT_DEPARTMENT"
const USER_PERMISSION_DELETE_DEPARTMENT = "DELETE_DEPARTMENT"
const USER_PERMISSION_REMOVE_DEPARTMENT = "REMOVE_DEPARTMENT"

const PARAMS_COMPANY_ID = "CompanyID"
const PARAMS_SYSTEM = "SYSTEM"
const PARAMS_DEPT_ID = "DepartmentID"
const PARAMS_STATUS = "Status"
const PARAMS_SK = "SK"
const PARAMS_PK = "PK"
const PARAMS_EMPTY = ""

// Upload type
const UPLOAD_TYPE_IMG = "img"
const UPLOAD_TYPE_FILE = "files"

// Image Types
const IMAGE_TYPE_USER = "user"
const IMAGE_TYPE_COMPANY = "company"
const IMAGE_TYPE_DEPARTMENT = "department"
const IMAGE_TYPE_INTEGRATION = "integration"
const IMAGE_TYPE_GROUP = "group"
const IMAGE_TYPE_STATIC = "static"

// Image prefix
const IMAGE_SUFFIX_ORIGINAL = "%s"
const IMAGE_SUFFIX_ORIGINAL_HOSTED = "%25s"
const IMAGE_SUFFIX_1024 = "_1024"
const IMAGE_SUFFIX_500 = "_500"
const IMAGE_SUFFIX_100 = "_100"

// Image size
const IMAGE_SIZE_1024 = 1024
const IMAGE_SIZE_500 = 500
const IMAGE_SIZE_100 = 100

// Config variables
const CONFIG_AWS_BUCKET = "aws.bucket"
const CONFIG_AWS_REGION = "aws.region"
const CONFIG_AWS_ENDPOINT_LOCAL = "aws.endpoint.local"
const CONFIG_AWS_ENDPOINT_DEV = "aws.endpoint.dev"
const CONFIG_AWS_ENDPOINT_PROD = "aws.endpoint.prod"
const CONFIG_MAIL_HOST = "mail.host"
const CONFIG_MAIL_PORT = "mail.port"
const CONFIG_MAIL_USERNAME = "mail.username"
const CONFIG_MAIL_PASSWORD = "mail.password"
const CONFIG_MAIL_FROM = "mail.from"

// Presign
const PRESIGN_DURATION = 60

// Log Type
// Check if other constants can be used instead of declaring LOG_ACTION_*
// Auth
const LOG_ACTION_SIGN_UP = "SIGN_UP"
const LOG_ACTION_ESTABLISH_COMPANY = "ESTABLISH_COMPANY"
const LOG_ACTION_SIGN_OUT = "SIGN_OUT"
const LOG_ACTION_SIGN_IN = "SIGN_IN"
const LOG_ACTION_REQUEST_PASSWORD_RESET = "REQUEST_PASSWORD_RESET"
const LOG_ACTION_PASSWORD_CHANGED = "PASSWORD_CHANGED"
const LOG_ACTION_ACTIVATE_ACCOUNT = "ACTIVATE_ACCOUNT"

// User
const LOG_ACTION_UPDATE_USER = "UPDATE_USER"

// Company
const LOG_ACTION_ADD_COMPANY = "ADD_COMPANY"
const LOG_ACTION_UPDATE_COMPANY = "UPDATE_COMPANY"
const LOG_ACTION_ADD_COMPANY_MEMBERS = "ADD_COMPANY_MEMBERS"
const LOG_ACTION_REMOVE_COMPANY_MEMBERS = "REMOVE_COMPANY_MEMBERS"
const LOG_ACTION_RESTORE_COMPANY_MEMBERS = "RESTORE_COMPANY_MEMBERS"
const LOG_ACTION_PERMANENTLY_REMOVE_COMPANY_MEMBERS = "PERMANENTLY_REMOVE_COMPANY_MEMBERS"
const LOG_ACTION_ADD_BOOKMARK_COMPANY = "ADD_BOOKMARK_COMPANY"
const LOG_ACTION_DELETE_BOOKMARK_COMPANY = "REMOVE_BOOKMARK_COMPANY"
const LOG_ACTION_REMOVE_COMPANY_MEMBERS_INTEGRATION_ACCESS = "REMOVE_COMPANY_MEMBERS_INTEGRATION_ACCESS"

const LOG_ACTION_INVITE_REMOVE_COMPANY_MEMBER = "INVITE_REMOVE_COMPANY_MEMBER"

const WIZARD_STATUS_DONE = "DONE"

// Subscription
const LOG_ACTION_CREATE_SUBSCRIPTION = "CREATE_SUBSCRIPTION"
const LOG_ACTION_UPDATE_SUBSCRIPTION = "UPDATE_SUBSCRIPTION"

// Group
const LOG_ACTION_ADD_GROUP = "ADD_GROUP"
const LOG_ACTION_ADD_BOOKMARK_GROUP = "ADD_BOOKMARK_GROUP"
const LOG_ACTION_DELETE_BOOKMARK_GROUP = "REMOVE_BOOKMARK_GROUP"
const LOG_ACTION_UPDATE_GROUP = "UPDATE_GROUP"
const LOG_ACTION_RENAME_GROUP = "RENAME_GROUP"
const LOG_ACTION_DELETE_GROUP = "REMOVE_GROUP"
const LOG_ACTION_ADD_GROUP_MEMBERS = "ADD_GROUP_MEMBERS"
const LOG_ACTION_ADD_GROUP_INTEGRATION = "ADD_GROUP_INTEGRATION"
const LOG_ACTION_REMOVE_GROUP_INTEGRATION = "REMOVE_GROUP_INTEGRATION"
const LOG_ACTION_REMOVE_GROUP_MEMBERS = "REMOVE_GROUP_MEMBERS"
const LOG_ACTION_REMOVE_INDIVIDUAL_MEMBERS = "REMOVE_INDIVIDUAL_MEMBERS"
const LOG_ACTION_CLONE_USER_GROUPS = "CLONE_USER_GROUPS"
const LOG_ACTION_BRANCH_GROUP = "BRANCH_GROUP"
const LOG_ACTION_CLONE_GROUP = "CLONE_GROUP"
const LOG_ACTION_MERGE_GROUP = "MERGE_GROUP"
const LOG_ACTION_REMOVE_GROUP_USER = "REMOVE_GROUP_USER"

// Department
const LOG_ACTION_ADD_DEPARTMENT = "ADD_DEPARTMENT"
const LOG_ACTION_UPDATE_DEPARTMENT = "UPDATE_DEPARTMENT"
const LOG_ACTION_DELETE_DEPARTMENT = "REMOVE_DEPARTMENT"

// Integration
const LOG_ACTION_CONNECT_INTEGRATION = "CONNECT_INTEGRATION"
const LOG_ACTION_DISCONNECT_INTEGRATION = "DISCONNECT_INTEGRATION"
const LOG_ACTION_REQUEST_INTEGRATION = "REQUEST_INTEGRATION"

// Help Center
const LOG_ACTION_SHARE_FEEDBACK = "SHARE_FEEDBACK"

// Role
const LOG_ACTION_ADD_ROLE = "ADD_ROLE"
const LOG_ACTION_UPDATE_ROLE = "UPDATE_ROLE"
const LOG_ACTION_DELETE_ROLE = "REMOVE_ROLE"
const LOG_ACTION_ASSIGN_ROLE = "ASSIGN_ROLE"
const LOG_ACTION_UNASSIGN_ROLE = "UNASSIGN_ROLE"

// Billing
const LOG_ACTION_MANAGE_BILLING = "MANAGE_BILLING"

// PERMISSIONS
// Company Policies
const EDIT_COMPANY = "EDIT_COMPANY"
const REMOVE_COMPANY = "REMOVE_COMPANY"

// Company Integration Policies
const ADD_INTEGRATION = "ADD_INTEGRATION"
const REMOVE_INTEGRATION = "REMOVE_INTEGRATION"
const CONNECT_INTEGRATION = "CONNECT_INTEGRATION"
const DISCONNECT_INTEGRATION = "DISCONNECT_INTEGRATION"
const ADD_SUB_INTEGRATION = "ADD_SUB_INTEGRATION"

// Company Member Policies
const ADD_COMPANY_MEMBER = "ADD_COMPANY_MEMBER"
const EDIT_COMPANY_MEMBER = "EDIT_COMPANY_MEMBER"
const REMOVE_COMPANY_MEMBER = "REMOVE_COMPANY_MEMBER"

// Department Policies
const ADD_DEPARTMENT = "ADD_DEPARTMENT"
const EDIT_DEPARTMENT = "EDIT_DEPARTMENT"
const REMOVE_DEPARTMENT = "REMOVE_DEPARTMENT"

// Group Policies
const ADD_GROUP = "ADD_GROUP"
const EDIT_GROUP = "EDIT_GROUP"
const REMOVE_GROUP = "REMOVE_GROUP"
const CLONE_GROUP = "CLONE_GROUP"
const MERGE_GROUP = "MERGE_GROUP"
const BRANCH_GROUP = "BRANCH_GROUP"

// Group Member Policies
const ADD_GROUP_MEMBER = "ADD_GROUP_MEMBER"
const REMOVE_GROUP_MEMBER = "REMOVE_GROUP_MEMBER"

// Group Integration Policies
const ADD_GROUP_INTEGRATION = "ADD_GROUP_INTEGRATION"
const REMOVE_GROUP_INTEGRATION = "REMOVE_GROUP_INTEGRATION"

// Role Management Policies
const ADD_ROLE = "ADD_ROLE"
const EDIT_ROLE = "EDIT_ROLE"
const REMOVE_ROLE = "REMOVE_ROLE"
const ASSIGN_ROLE = "ASSIGN_ROLE"
const UNASSIGN_ROLE = "UNASSIGN_ROLE"

// Billing Policies
const MANAGE_BILLING = "MANAGE_BILLING"

const TOTAL_ROLE_COUNT = 28

// =========
const REMOVE_MEMBER = "REMOVE_MEMBER"

//BOOL

const BOOL_TRUE = "TRUE"
const BOOL_FALSE = "FALSE"

//COLOR PALLETE
//https://flatuicolors.com/palette/au

var COLORS = map[int]string{
	1:  "#ffbe76",
	2:  "#ff7979",
	3:  "#badc58",
	4:  "#f9ca24",
	5:  "#f0932b",
	6:  "#eb4d4b",
	7:  "#6ab04c",
	8:  "#7ed6df",
	9:  "#e056fd",
	10: "#686de0",
	11: "#30336b",
	12: "#95afc0",
	13: "#22a6b3",
	14: "#be2edd",
	15: "#e056fd",
	16: "#e056fd",
}

// STRIPE PRICES

// montly subscription: $5 per seat
const MONTHLY_SUBSCRIPTION = "price_1JbvqlAtcfjD5l2xsC391aNP"

// yearly subscription: $48 per seat
const YEARLY_SUBSCRIPTION = "price_1JcS94AtcfjD5l2xQR2OASWf"

// montly subscription: $5 per seat
const TEST_DAILY = "price_1JhQCYAtcfjD5l2xULrc3Z4R"

// yearly subscription: $48 per seat
const TEST_YEARLY = "price_1JhQD9AtcfjD5l2xA583vpGw"

// subscription type
const FREE_TIER = "FREE"

const FREE_TRIAL = "FREE-TRIAL"

// JOB NAMES
const JOB_DELETE_GOOGLE_ACCOUNT = "DELETE_GOOGLE_ACCOUNT"

// CRON JOB NAMES
const JOB_GROUP_MEMBERS = "JOB_GROUP_MEMBERS"
const JOB_USER = "JOB_USER"

// CRON JOB TYPES
const JOB_ADD_GROUP_MEMBERS = "JOB_ADD_GROUP_MEMBERS"
const JOB_REMOVE_GROUP_MEMBERS = "JOB_REMOVE_GROUP_MEMBERS"
const JOB_MOVE_GROUP_MEMBERS = "JOB_MOVE_GROUP_MEMBERS"
const JOB_ON_BOARDING_USER = "JOB_ON_BOARDING_USER"
const JOB_OFF_BOARDING_USER = "JOB_OFF_BOARDING_USER"

// INTEGRATION SLUGS
const INTEG_SLUG_GOOGLE_CLOUD = "google-cloud"
const INTEG_SLUG_GOOGLE_ADMIN = "google-admin"
const INTEG_SLUG_GOOGLE_DRIVE = "google-drive"
const INTEG_SLUG_GCP_PROJECTS = "google-cloud"
const INTEG_SLUG_FIREBASE = "firebase"

const INTEG_SLUG_OFFICE_365 = "office365"
const INTEG_SLUG_365_GROUPS = "ms-365-groups"
const INTEG_SLUG_AZURE_ADMIN = "azure-admin"
const INTEG_SLUG_ONEDRIVE = "one-drive"
const INTEG_SLUG_SHAREPOINT = "ms-sharepoint"
const INTEG_SLUG_LICENSE = "ms-office"

const INTEG_SLUG_SALESFORCE = "salesforce"
const INTEG_SLUG_SALESFORCE_PERMISSION_SETS = "salesforce-permission-sets"
const INTEG_SLUG_SALESFORCE_CHATTER = "salesforce-chatter"

const INTEG_SLUG_AWS = "aws"
const INTEG_SLUG_IAM = "iam"

const INTEG_SLUG_BITBUCKET = "bitbucket"
const INTEG_SLUG_BITBUCKET_GROUPS = "bitbucket-groups"
const INTEG_SLUG_BITBUCKET_PROJECTS = "bitbucket-projects"
const INTEG_SLUG_BITBUCKET_REPOSITORIES = "bitbucket-repositories"

const INTEG_SLUG_JIRA = "jira"
const INTEG_SLUG_JIRA_USER = "jira-user"
const INTEG_SLUG_JIRA_PROJECT = "jira-project"

const INTEG_SLUG_GITHUB = "github"
const INTEG_SLUG_GITHUB_TEAM = "github-team"
const INTEG_SLUG_GITHUB_REPO = "github-repo"

const INTEG_SLUG_DROPBOX = "dropbox"
const INTEG_SLUG_DROPBOX_GROUP = "dropbox-group"
const INTEG_SLUG_DROPBOX_FOLDER = "dropbox-folder"

const INTEG_SLUG_ZOOM = "zoom"

const INTEG_SLUG_ZENDESK = "zendesk"
const INTEG_SLUG_ZENDESK_GROUPS = "zendesk-groups"
const INTEG_SLUG_ZENDESK_TICKETS = "zendesk-tickets"

const INTEG_SLUG_SLACK = "slack"
const INTEG_SLUG_SLACK_CHANNEL = "slack-channel"
const INTEG_SLUG_SLACK_USERGROUPS = "slack-usergroups"

const INTEG_SLUG_HOOTSUITE = "hootsuite"
const INTEG_SLUG_HOOTSUITE_TEAMS = "hootsuite-teams"

const INTEG_SLUG_DOCUSIGN = "docusign"
const INTEG_SLUG_DOCUSIGN_USERGROUPS = "docusign-usergroups"

const INTEG_SLUG_ASANA = "asana"
const INTEG_SLUG_ASANA_TEAMS = "asana-teams"
const INTEG_SLUG_ASANA_PROJECTS = "asana-projects"

const INTEG_SLUG_TRELLO = "trello"
const INTEG_SLUG_TRELLO_WORKSPACES = "trello-workspaces"
const INTEG_SLUG_TRELLO_BOARDS = "trello-boards"

const INTEG_SLUG_AZURE_DEVOPS = "azure-devops"
const INTEG_SLUG_AZURE_DEVOPS_PROJECTS = "azure-devops-projects"

const INTEG_SLUG_GITLAB = "gitlab"
const INTEG_SLUG_GITLAB_GROUPS = "gitlab-groups"
const INTEG_SLUG_GITLAB_PROJECTS = "gitlab-projects"

const INTEG_SLUG_POWER_BI = "power-bi"
const INTEG_SLUG_POWER_BI_GROUPS = "power-bi-groups"

const INTEG_SLUG_ORACLE = "oracle"
const INTEG_SLUG_ORACLE_AUTONOMOUS_DB = "oracle-autonomous-database"

// Settings
const DEFAULT_EMAIL_ADDRESS_FORMAT = "FIRSTNAME.LASTNAME"

var AUTOMATED_EMAIL_ADDRESS_FORMATS = [...]string{
	"FIRSTNAME.LASTNAME",
	"LASTNAME.FIRSTNAME",
	"FIRSTNAME_LASTNAME",
	"LASTNAME_FIRSTNAME",
	"FIRSTNAME-INITIAL_LASTNAME",
	"FIRSTNAME_LASTNAME-INITIAL",
	"RANDOM",
}

// jira integration
const JIRA_UNASSIGNED = "UNASSIGNED"
const JIRA_PROJECT_TYPE_KEY_SOFTWARE = "software"
const JIRA_PROJECT_TEMPLATE_KEY = "com.pyxis.greenhopper.jira:gh-simplified-scrum-classic"

const ERROR_REASON_DUPLICATE_EMAIL_ADDRESS = "DUPLICATE_EMAIL_ADDRESSES"
const ERROR_REASON_EMAIL_ADDRESSES_EXISTS = "EMAIL_ADDRESSES_EXISTS"

// STRIPE
const BILLING_TYPE_MONTHLY = "MONTHLY"
const BILLING_TYPE_YEARLY = "YEARLY"

const STRIPE_DISCOUNT_ENABLE = true

// Test
const STRIPE_MONTHLY_PRICE_ID = "price_1Kl7laAtcfjD5l2xuTlSK8TP"
const STRIPE_YEARLY_PRICE_ID = "price_1Kl7o2AtcfjD5l2xMKC2tNw2"
const STRIPE_20_DISCOUNT_ID = "HDTl4hbu"

var PRICE = map[string]int{
	"price_1Kl7laAtcfjD5l2xuTlSK8TP": 5, //user / monthly
	"price_1Kl7o2AtcfjD5l2xMKC2tNw2": 4, //user / year
}

// Prod
// const STRIPE_MONTHLY_PRICE_ID = "price_1Kpm4pAtcfjD5l2xcH9EGuYW"
// const STRIPE_YEARLY_PRICE_ID = "price_1Kpm4zAtcfjD5l2xSBKxAh06"
// const STRIPE_20_DISCOUNT_ID = "XiL2kFQM"

// var PRICE = map[string]int{
// 	"price_1Kpm4pAtcfjD5l2xcH9EGuYW": 5, //user / monthly
// 	"price_1Kpm4zAtcfjD5l2xSBKxAh06": 4, //user / year
// }

// includes in services
const ITEM_INCLUDE_GROUPS = "groups"
const ITEM_INCLUDE_GROUPS_MEMBERS = "groups,members"
const ITEM_INCLUDE_GROUPS_INTEGRATIONS = "groups,integrations"

// ROLE ID
const ROLE_ID_COMPANY_ADMIN = "52e62ee3-a0db-4e32-b1ba-3b932ffd8f5e"
const ROLE_ID_DEPARTMENT_ADMIN = "0094b3ec-48e9-4679-bbf5-ddd6cc374c27"
const ROLE_ID_GROUP_ADMIN = "86737da9-27c5-4636-a301-fc73613ea27a"

// Origin
const ORIGIN_DEFAULT = "SaaSConsole"
const ORIGIN_GOOGLE = "Google"
const ORIGIN_OFFICE = "Microsoft"

// ActionItemTypes
const ACTION_ITEM_TYPE_CREATE_DEPARTMENT = "CREATE_DEPARTMENT"
const ACTION_ITEM_TYPE_CREATE_USER = "CREATE_USER"
const ACTION_ITEM_TYPE_ADD_USERS_TO_GROUP = "ADD_USERS_TO_GROUP"
const ACTION_ITEM_TYPE_ADD_REMOVE_USER = "ADD_REMOVE_USER"

const ACTION_ITEM_TYPE_NEW_INTEGRATION = "NEW_INTEGRATION"
const ACTION_ITEM_TYPE_REMOVE_USER_TO_COMPANY = "REMOVE_USER_TO_COMPANY"
const ACTION_ITEM_TYPE_INVITE_REMOVE_COMPANY_MEMBER = "INVITE_REMOVE_COMPANY_MEMBER"

const ACTION_ITEM_TYPE_SUGGEST_INTEGRATION_CATEGORY = "SUGGEST_INTEGRATION_CATEGORY"
const ACTION_ITEM_TYPE_IMPORT_INTEGRATION_CONTACTS = "IMPORT_INTEGRATION_CONTACTS"
const ACTION_ITEM_TYPE_SUGGEST_HOT_INTEGRATION = "SUGGEST_HOT_INTEGRATION"

// Priorities
const PRIORITY_HIGH = "HIGH"
const PRIORITY_MEDIUM = "MEDIUM"
const PRIORITY_LOW = "LOW"

// Source Types
const SOURCE_TYPE_SYSTEM = "System"
const SOURCE_TYPE_INTEGRATION = "Integration"

// User Types
const USER_TYPE_COMPANY_OWNER = "COMPANY_OWNER"
const USER_TYPE_COMPANY_MEMBER = "COMPANY_MEMBER"

// Integration Badges
const INTEGRATION_BADGE_HOT_PERCENTAGE = "0.75"
const INTEGRATION_BADGE_NEW_DAYS = "30"

// New Log Events
const GOOGLE_INVITE_USER = "GOOGLE:INVITE_USER"
const GOOGLE_REMOVE_USER = "GOOGLE:REMOVE_USER"
const GOOGLE_ADMIN_UPDATE_PERMISSIONS = "GOOGLE:ADMIN_UPDATE_PERMISSIONS"
const GOOGLE_ADMIN_CREATE = "GOOGLE:ADMIN_CREATE"
const GOOGLE_ADMIN_UPDATE = "GOOGLE:ADMIN_UPDATE"
const GOOGLE_ADMIN_DISCONNECT = "GOOGLE:ADMIN_DISCONNECT"
const GOOGLE_ADMIN_ADD_MEMBER = "GOOGLE:ADMIN_ADD_MEMBER"
const GOOGLE_ADMIN_REMOVE_MEMBER = "GOOGLE:ADMIN_REMOVE_MEMBER"
const GOOGLE_DRIVE_CREATE = "GOOGLE:DRIVE_CREATE"
const GOOGLE_DRIVE_UPDATE = "GOOGLE:DRIVE_UPDATE"
const GOOGLE_DRIVE_DISCONNECT = "GOOGLE:DRIVE_DISCONNECT"
const GOOGLE_DRIVE_ADD_MEMBER = "GOOGLE:DRIVE_ADD_MEMBER"
const GOOGLE_DRIVE_MEMBER_ROLE_UPDATE = "GOOGLE:DRIVE_MEMBER_ROLE_UPDATE"
const GOOGLE_GCP_CREATE = "GOOGLE:GCP_CREATE"
const GOOGLE_GCP_UPDATE = "GOOGLE:GCP_UPDATE"
const GOOGLE_GCP_DISCONNECT = "GOOGLE:GCP_DISCONNECT"
const GOOGLE_GCP_ADD_MEMBER = "GOOGLE:GCP_ADD_MEMBER"
const GOOGLE_GCP_REMOVE_MEMBER = "GOOGLE:GCP_REMOVE_MEMBER"
const GOOGLE_GCP_MEMBER_ROLE_UPDATE = "GOOGLE:GCP_MEMBER_ROLE_UPDATE"
const GOOGLE_FIREBASE_CREATE = "GOOGLE:FIREBASE_CREATE"
const GOOGLE_FIREBASE_UDPATE = "GOOGLE:FIREBASE_UDPATE"
const GOOGLE_FIREBASE_DISCONNECT = "GOOGLE:FIREBASE_DISCONNECT"
const GOOGLE_FIREBASE_ADD_MEMBER = "GOOGLE:FIREBASE_ADD_MEMBER"
const GOOGLE_FIREBASE_REMOVE_MEMBER = "GOOGLE:FIREBASE_REMOVE_MEMBER"
const GOOGLE_FIREBASE_MEMBER_ROLE_UPDATE = "GOOGLE:FIREBASE_MEMBER_ROLE_UPDATE"

// Notification Types
const REQUEST_PERMISSION_UPDATE = "REQUEST_PERMISSION_UPDATE"
const REQUEST_COMPANY_ROLE_UPDATE = "REQUEST_COMPANY_ROLE_UPDATE"
const REQUEST_REMOVE_INTEGRATION = "REQUEST_REMOVE_INTEGRATION"
const REQUEST_CONNECT_INTEGRATION = "REQUEST_CONNECT_INTEGRATION"
const REQUEST_DISCONNECT_INTEGRATION = "REQUEST_DISCONNECT_INTEGRATION"
const REQUEST_TO_JOIN_GROUP = "REQUEST_TO_JOIN_GROUP"
const REQUEST_STATUS_UPDATE = "REQUEST_STATUS_UPDATE"
const REQUEST_COMPANY_ROLE_ACCEPT = "REQUEST_COMPANY_ROLE_ACCEPT"
const REQUEST_TO_CREATE_ACCOUNT = "REQUEST_TO_CREATE_ACCOUNT"
const REQUEST_TO_MATCH_ACCOUNT = "REQUEST_TO_MATCH_ACCOUNT"
const ROLE_UPDATE = "ROLE_UPDATE"

// Pre-made Roles
var PRE_MADE_ROLES = []string{
	"company admin",
	"department admin",
	"group admin",
	"group member",
}

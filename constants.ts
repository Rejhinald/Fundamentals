export const APP = {
  VERSION: 'v1.0.0'
};

export const AUTH = {
  PASSWORD_LENGTH: 8
};

 // todo make dynamic
export const BILLING = {
  FREE_USER: 5,
  // PER_SEAT: 5,
  PER_SEAT: 0.5,
  // PRICE: 5,
  PRICE: 0.5,
  DISCOUNT: 0.2,
  MAX_SEATS: 10000,
  MIN_SEATS: 5,
  TRIALING_MAX_SEATS: 100,
}

export const MODAL_DATA_DELETE_TYPE = {
  GROUPS: 'GROUPS',
  DEPARTMENTS: 'DEPARTMENTS',
  COMPANIES: 'COMPANIES',
  PROFILE: 'PROFILES',
  GOOGLE: 'GOOGLE ACCOUNT',
  OFFICE: 'Office 365 ACCOUNT',
  HOOTSUITE: 'Hootsuite Account',
  JIRA: 'JIRA ACCOUNT',
  DROPBOX: 'DROPBOX ACCOUNT',
  DOCUSIGN: 'Docusign Account',
  ASANA: 'Asana Account',
  ZENDESK: 'Zendesk Account',
}

export const MODAL_DATA_TYPE = {
  GROUP: 'GROUP',
  DEPARTMENT: 'DEPARTMENT',
  COMPANY: 'COMPANY',
  PROFILE: 'PROFILE',
  GOOGLE: 'GOOGLE ACCOUNT',
  OFFICE: 'Office 365 ACCOUNT',
  HOOTSUITE: 'Hootsuite Account',
  JIRA: 'JIRA ACCOUNT',
  DROPBOX: 'DROPBOX ACCOUNT',
  DOCUSIGN: 'Docusign Account',
  ASANA: 'Asana Account',
  ZENDESK: 'Zendesk Account',
}

export const ACTION = {
  ADD: 'ADD',
  UPDATE: 'UPDATE',
  DELETE: 'DELETE',
  REMOVE: 'REMOVE',
  MOVE: 'MOVE',
  MERGE: 'MERGE',
  CLONE: 'CLONE',
  BRANCH: 'BRANCH',
};

export const ACTIVE_PAGE_VIEW = {

}

export const ITEM_STATUS = {
  ACTIVE: "ACTIVE",
  DEFAULT: "DEFAULT",
  INACTIVE: "INACTIVE",
  PENDING: "PENDING",
  CANCEL: "CANCEL",
  DELETED: "DELETED",
  DEACTIVATED: "DEACTIVATED",
  PERMANENTLY_DELETED: "PERMANENTLY_DELETED",
  NOT_SYNCED: "NOT_SYNCED",
  NOT_LINKED: "NOT LINKED",
  UNREGISTERED: "UNREGISTERED",
  NOT_SENT: "NOT SENT",
  INVITE_SENT:"INVITE SENT",
  NO_LICENSE: "NO_LICENSE",
  SUSPENDED: "SUSPENDED",
  NONEXISTENT: "NONEXISTENT",
  NOTVERIFIED: "NOTVERIFIED",
  NOT_EXISTING: "NOT_EXISTING",
  NOT_INVITED: "NOT_INVITED",
  BLOCKED: "BLOCKED",
  BANNED: "BANNED",
  PRIVATE: "PRIVATE",
  PUBLIC: "PUBLIC",
}

export const INTEGRATION_STATUS = {
  ACTIVE: "ACTIVE",
  INACTIVE: "INACTIVE",
  PENDING: "PENDING",
  COMING_SOON: "COMING_SOON",
  IN_DEVELOPMENT: "IN_DEVELOPMENT",
  UNDER_MAINTENANCE: "UNDER_MAINTENANCE",
  SUSPENDED: "SUSPENDED",
}


export const USER_ITEM_STATUS = [ITEM_STATUS.ACTIVE, ITEM_STATUS.INACTIVE, ITEM_STATUS.PENDING, ITEM_STATUS.DELETED];

export const ENTITY_TYPE = {
  AUTH: 'AUTH',
  USER: 'USER',
  COMPANY: 'COMPANY',
  DEPARTMENT: 'DEPARTMENT',
  GROUP: 'GROUP',
  GROUPMEMBER: 'GROUPMEMBER',
  COMPANYMEMBER: 'COMPANYMEMBER',
  DEPARTMENTMEMBER: 'DEPARTMENTMEMBER',
  SUBSCRIPTION: 'SUBSCRIPTION',
  LOG: 'LOG',
  INTEGRATION: 'INTEGRATION',
  ROLE: 'ROLE',
};

export const HTTP_STATUS = {
  // Standard HTTP Status Codes
  _200: '200',
  _201: '201',
  _202: '202',
  _203: '203',
  _204: '204',

  _400: '400',
  _401: '401',
  _403: '403',
  _404: '404',
  _405: '405',
  _409: '409',
  _422: '422',
  _500: '500',

  // Custom HTTP Status Codes
  _460: '460',
  _461: '461',
  _462: '462',
  _463: '463',
  _464: '464',
  _465: '465',
  _466: '466',
  _467: '467',
  _468: '468',
  _469: '469',
  _470: '470',
  _471: '471',
  _472: '472',
  _473: '473',
  _475: '475',
  _476: '476',
  _477: '477',
  _489: '489',
  _485: '485',
  _490: '490',
  _492: '492',
  _493: '493',
};

export const QUERY_LIMIT = 50;

export const INTEG_SLUG = {
  GOOGLE_CLOUD: {
    SELF: "google-cloud",
    FIREBASE: "firebase",
    GOOGLE_ADMIN: "google-admin",
    GOOGLE_DRIVE: "google-drive",
    GOOGLE_CLOUD: "google-cloud",
    GCP: "google-cloud", // TODO change to gcp
  },

  OFFICE: {
    SELF: "office365",
    LICENSE: "ms-office",
    ONE_DRIVE: "one-drive",
    SHAREPOINT: "ms-sharepoint",
    MS_365_GROUPS: "ms-365-groups",
  },

  JIRA: {
    SELF: "jira",
    USERS: "jira-user",
    PROJECTS: "jira-project"
  },

  AWS: {
    SELF: "aws",
    IAM: "iam"
  },

  GITHUB: {
    SELF: "github",
    TEAMS: "github-team",
    REPO: "github-repo",
  },

  DROPBOX: {
    SELF: "dropbox",
    GROUP: "dropbox-group",
    FOLDER: "dropbox-folder"
  },

  SALESFORCE: {
    SELF: "salesforce",
    CHATTER: "salesforce-chatter",
    PERMISSION_SETS: "salesforce-permission-sets",
  },

  ZOOM: {
    SELF: "zoom",
    MEETING: "zoom-meeting",
    GROUP: "zoom-group",
    ROLE: "zoom-role",
    WEBINAR: "zoom-webinar",
  },

  ZENDESK: {
    SELF: "zendesk",
    GROUPS: "zendesk-groups",
    TICKETS: "zendesk-tickets",
  },

  SLACK: {
    SELF: "slack",
    SLACK_CHANNEL_MANAGEMENT: "slack-channel",
    SLACK_USERGROUPS: "slack-usergroups",
  },

  HOOTSUITE: {
    SELF: "hootsuite",
    HOOTSUITE_TEAMS: "hootsuite-teams",
  },

  DOCUSIGN: {
    SELF: "docusign",
    GROUPS: "docusign-groups",
  },

  ASANA: {
    SELF: "asana",
    TEAMS: "asana-teams",
    PROJECTS: "asana-projects"
  },

  TRELLO: {
    SELF: "trello",
    WORKSPACES: "trello-workspaces",
    BOARDS: "trello-boards"
  },

  BITBUCKET: {
    SELF: "bitbucket",
    GROUPS: "bitbucket-groups",
    PROJECTS: "bitbucket-projects",
    REPOSITORIES: "bitbucket-repositories",
  },

  AZURE_DEVOPS: {
    SELF: "azure-devops",
    PROJECTS: "azure-devops-projects"
  },

  GITLAB: {
    SELF: "gitlab",
    GROUPS: "gitlab-groups",
    PROJECTS: "gitlab-projects"
  },
  
  POWER_BI: {
    SELF: "power-bi",
    GROUPS: "power-bi-groups"
  },
  ORACLE: {
    SELF: "oracle",
    AUTONOMOUS_DATABASE: "oracle-autonomous-database",
    MYSQL_HEATWAVE: "oracle-mysql-heatwave",
    POSTGRESQL: "oracle-postgresql",
    NOSQL: "oracle-nosql",
  },
  
};

export const PREFIX = {
  USER: 'USER#',
  COMPANY: 'COMPANY#',
  DEPARTMENT: 'DEPARTMENT#',
  GROUP: 'GROUP#',
  SUBSCRIPTION: 'SUBSCRIPTION#',
  LOG: 'LOG#',
};

export const REGEX = {
  LOWER_CASE_LETTERS: /[a-z]/g,
  NUMBERS: /[0-9]/g,
  NUMBERS_ONLY: /^[0-9]*$/g,
  SPECIAL_CHARACTERS: /[-!@#$%^&*()_+|~=`{}\[\]:";'<>?,.\/]/,
  UPPER_CASE_LETTERS: /[A-Z]/g,
  SLACK_CREATE: /[^!@#$%^&*()+|~=`{}\[\]:";'<>?,.\/]+/,
  USER_COUNT_REGEX: /^([1-9][0-9]{0,3}|10000)$/,
  USER_COUNT_TRIALING_REGEX: /^[1-9]$|^[1-9][0-9]$|^(100)$/
};

export const USER_PERMISSION = {
  ADD_PEOPLE: 'ADD_PEOPLE',
  DELETE_PEOPLE: 'DELETE_PEOPLE',
  EDIT_PEOPLE: 'EDIT_PEOPLE',
  ADD_GROUP: 'ADD_GROUP',
  EDIT_GROUP: 'EDIT_GROUP',
  DELETE_GROUP: 'DELETE_GROUP',
  CLONE_GROUP: 'CLONE_GROUP',
  BRANCH_GROUP: 'BRANCH_GROUP',
  MERGE_GROUP: 'MERGE_GROUP',
  ADD_COMPANY: 'ADD_COMPANY',
  EDIT_COMPANY: 'EDIT_COMPANY',
  DELETE_COMPANY: 'DELETE_COMPANY',
  ADD_DEPARTMENT: 'ADD_DEPARTMENT',
  EDIT_DEPARTMENT: 'EDIT_DEPARTMENT',
  DELETE_DEPARTMENT: 'DELETE_DEPARTMENT',

};

export const CATEGORY_PERMISSION = {
  COMPANY: 'COMPANY_PERMISSIONS',
  COMPANY_INTEGRATION: 'COMPANY_INTEGRATION_PERMISSIONS',
  COMPANY_MEMBER: 'COMPANY_MEMBER_PERMISSIONS',
  DEPARTMENT: 'DEPARTMENT_PERMISSIONS',
  GROUP: 'GROUP_PERMISSIONS',
  GROUP_INTEGRATION: 'GROUP_INTEGRATION_PERMISSIONS',
  GROUP_MEMBER: 'GROUP_MEMBER_PERMISSIONS',
  ROLES: 'ROLE_PERMISSIONS',
  BILLING: 'BILLING_PERMISSIONS'
}

export const CATEGORY_PERMISSIONS_ARRAY = [
  {
    CODE: 'COMPANY_PERMISSIONS',
    NAME: 'Company'
  },
  {
    CODE: 'COMPANY_INTEGRATION_PERMISSIONS',
    NAME: 'Company Integration'
  },
  {
    CODE: 'COMPANY_MEMBER_PERMISSIONS',
    NAME: 'Company Member'
  },
  {
    CODE: 'DEPARTMENT_PERMISSIONS',
    NAME: 'Department'
  },
  {
    CODE: 'GROUP_PERMISSIONS',
    NAME: 'Groups'
  },
  {
    CODE: 'GROUP_INTEGRATION_PERMISSIONS',
    NAME: 'Group Integrations'
  },
  {
    CODE: 'GROUP_MEMBER_PERMISSIONS',
    NAME: 'Group Members'
  },
  {
    CODE: 'ROLE_PERMISSIONS',
    NAME: 'Roles'
  },
  {
    CODE: 'BILLING_PERMISSIONS',
    NAME: 'Billing'
  }
]

export const USER_ROLE = {
  ADMIN: 'ADMIN',
  VIEWER: 'VIEWER',
  USER: 'USER'
};

export const ACTION_PRIORITY = {
  HIGH: 'HIGH',
  MEDIUM: 'MEDIUM',
  LOW: 'LOW'
}

export const AUTH_TOKEN_KEY = "authToken";
//Constants for filter all
export const FILTER_ALL = {
  DEPARTMENT: { value: undefined, label: 'All Departments' },
  GROUP: { value: undefined, label: 'All Groups' },
  MODULE: { value: undefined, label: 'All Modules' },
  ACTION: { value: undefined, label: 'All Actions' },
  PRIORITY: {value: undefined, label: 'All'},
  INTEGRATION: {value: "", label: 'All'},
  SUBINTEGRATION: {value: "", label: 'All'},
  LEVEL: {value: "", label: 'All Level'},
  SOURCE: { value: "", label: "All" }
}
// Options
export const actionOptions = [
  FILTER_ALL.ACTION,
  {
    value: ACTION.ADD,
    label: "Add",
  },
  {
    value: ACTION.REMOVE,
    label: "Remove",
  },
  {
    value: ACTION.UPDATE,
    label: "Update",
  },
]

export const moduleOptions = [
  FILTER_ALL.MODULE,
  {
    value: ENTITY_TYPE.USER,
    label: "User",
  },
  {
    value: ENTITY_TYPE.COMPANY,
    label: "Company",
  },
  {
    value: ENTITY_TYPE.DEPARTMENT,
    label: "Department",
  },
  {
    value: ENTITY_TYPE.GROUP,
    label: "Groups",
  },
]

export const sourceOptions = [
  FILTER_ALL.SOURCE,
  {
    value: "SYSTEM",
    label: "System",
  },
  {
    value: "INTEGRATION",
    label: "Integration",
  }
]

export const priorityOptions = [
  FILTER_ALL.PRIORITY,
  {
    value: ACTION_PRIORITY.HIGH,
    label: "High",
  },
  {
    value: ACTION_PRIORITY.MEDIUM,
    label: "Medium",
  },
  {
    value: ACTION_PRIORITY.LOW,
    label: "Low",
  },
]

export const levelOptions = [
  FILTER_ALL.LEVEL,
  {
    value: "INFORMATIONAL",
    label: "Info",
  },
  {
    value: "WARNING",
    label: "Warning",
  },
  {
    value: "FATAL",
    label: "Fatal",
  },
]

export const FILE_UPLOAD = {
  FILE_SIZE_LIMIT: 10000000,
  FILE_TYPE_REQUIRED: {
    IMAGE_PNG: "image/png",
    IMAGE_JPEG: "image/jpeg"
  },
  IMAGE_FILE_TYPES: ".png, .jpg, .jpeg"
}

export const IMAGE_SIZE = {
  _1024: "_1024",
  _500: "_500",
  _100: "_100",
}

export const ACTION_ITEM_TYPE = {
  CREATE_DEPARTMENT: "CREATE_DEPARTMENT",
  CREATE_USER: "CREATE_USER",
  ADD_USERS_TO_GROUP: "ADD_USERS_TO_GROUP",
  ADD_REMOVE_USER: "ADD_REMOVE_USER",
  SUGGEST_INTEGRATION_CATEGORY: "SUGGEST_INTEGRATION_CATEGORY",
  SUGGEST_HOT_INTEGRATION: "SUGGEST_HOT_INTEGRATION",
  IMPORT_INTEGRATION_CONTACTS: "IMPORT_INTEGRATION_CONTACTS",
  NEW_INTEGRATION: "NEW_INTEGRATION",
  REMOVE_USER_TO_COMPANY: "REMOVE_USER_TO_COMPANY",

  INVITE_REMOVE_COMPANY_MEMBER: "INVITE_REMOVE_COMPANY_MEMBER",
}

export const LOG_ACTION = {
  SIGN_UP: "SIGN_UP",
  ESTABLISH_COMPANY: "ESTABLISH_COMPANY",
  SIGN_OUT: "SIGN_OUT",
  SIGN_IN: "SIGN_IN",
  REQUEST_PASSWORD_RESET: "REQUEST_PASSWORD_RESET",
  PASSWORD_CHANGED: "PASSWORD_CHANGED",
  ACTIVATE_ACCOUNT: "ACTIVATE_ACCOUNT",

  ADD_COMPANY: "ADD_COMPANY",
  UPDATE_COMPANY: "UPDATE_COMPANY",

  ADD_COMPANY_MEMBERS: "ADD_COMPANY_MEMBERS",
  REMOVE_COMPANY_MEMBERS: "REMOVE_COMPANY_MEMBERS",
  INVITE_REMOVE_COMPANY_MEMBER: "INVITE_REMOVE_COMPANY_MEMBER",
  PERMANENTLY_REMOVE_COMPANY_MEMBERS: "PERMANENTLY_REMOVE_COMPANY_MEMBERS",
  RESTORE_COMPANY_MEMBERS: "RESTORE_COMPANY_MEMBERS",

  ADD_GROUP_MEMBERS: "ADD_GROUP_MEMBERS",
  DELETE_GROUP_MEMBERS: "DELETE_GROUP_MEMBERS",
  REMOVE_GROUP_MEMBERS: "REMOVE_GROUP_MEMBERS",
  REMOVE_INDIVIDUAL_MEMBERS: "REMOVE_INDIVIDUAL_MEMBERS",
  MOVE_GROUP_MEMBERS: 'MOVE_GROUP_MEMBERS',
  MERGE_GROUP: "MERGE_GROUP",
  ADD_GROUP: "ADD_GROUP",
  UPDATE_GROUP: "UPDATE_GROUP",
  DELETE_GROUP: "DELETE_GROUP",
  REMOVE_GROUP: "REMOVE_GROUP",
  CLONE_GROUP: "CLONE_GROUP",
  BRANCH_GROUP: "BRANCH_GROUP",
  ADD_GROUP_INTEGRATION: "ADD_GROUP_INTEGRATION",
  DELETE_BOOKMARK_GROUP: "DELETE_BOOKMARK_GROUP",
  REMOVE_BOOKMARK_GROUP: "REMOVE_BOOKMARK_GROUP",
  ADD_BOOKMARK_GROUP: "ADD_BOOKMARK_GROUP",

  UPDATE_USER: "UPDATE_USER",

  UPDATE: "UPDATE",
  CLONE_USER_GROUPS: "CLONE_USER_GROUPS",

  CONNECT_INTEGRATION: "CONNECT_INTEGRATION",
  DISCONNECT_INTEGRATION: "DISCONNECT_INTEGRATION",
  REQUEST_INTEGRATION: "REQUEST_INTEGRATION",

  ADD_ROLE: "ADD_ROLE",
  DELETE_ROLE: "DELETE_ROLE",
  REMOVE_ROLE: "REMOVE_ROLE",
  UPDATE_ROLE: "UPDATE_ROLE",

  ADD_DEPARTMENT: "ADD_DEPARTMENT",
  UPDATE_DEPARTMENT: "UPDATE_DEPARTMENT",
  DELETE_DEPARTMENT: "DELETE_DEPARTMENT",
  REMOVE_DEPARTMENT: "REMOVE_DEPARTMENT",

  REMOVE_GROUP_INTEGRATION: "REMOVE_GROUP_INTEGRATION",

  REMOVE_COMPANY_MEMBERS_INTEGRATION_ACCESS: "REMOVE_COMPANY_MEMBERS_INTEGRATION_ACCESS"

}

//Subscription Status
export const SUBSCRIPTION_STATUS = {
  EXPIRED: "EXPIRED",
  CANCELLED: "CANCELED",
  SUSPENDED: "SUSPENDED",
  FREE_TRIAL_EXPIRED: "FREE_TRIAL_EXPIRED",
  CARD_REJECTED: "CARD_REJECTED",
  ACTIVE: "ACTIVE",
  INCOMPLETE: "INCOMPLETE",
  INCOMPLETE_EXPIRED: "INCOMPLETE_EXPIRED",
  PAST_DUE: "PAST_DUE",
  UNPAID: "UNPAID",
  TRIALING: "TRIALING",
  NO_SUBSCRIPTION: "NO_SUBSCRIPTION",
}

////////////////////
// G SUITE
////////////////////

// Google Admin (Directory)
export const USERS_MAX_RESULT = "20";
export const GET_USERS_FIELD = "nextPageToken,users(id,name,primaryEmail,customerId)";
export const GROUP_ROLE = {
  MEMBER: "MEMBER",
}
export const GOOGLE_SCOPES = "https://www.googleapis.com/auth/admin.directory.user https://www.googleapis.com/auth/admin.directory.group https://www.googleapis.com/auth/cloudplatformprojects https://www.googleapis.com/auth/admin.directory.rolemanagement https://www.googleapis.com/auth/drive https://www.googleapis.com/auth/iam https://www.googleapis.com/auth/cloud-identity.userinvitations";

// GCP (Google Cloud Platform)
export const GCP_PERMISSIONS = {
  CREATE_PROJECT: "resourcemanager.projects.create",
  LISTS_PROJECT: "resourcemanager.projects.list",
  GET_PROJECT: "resourcemanager.projects.get",
  SET_IAM_POLICY: "resourcemanager.projects.setIamPolicy",
  FIREBASE_PROJECTS_UPDATE: "firebase.projects.update",
}

export const GCP_ROLES = {
  PROJECT_CREATOR: "roles/resourcemanager.projectCreator",
  FIREBASE_ADMIN: "roles/firebase.admin",
  PROJECT_IAM_ADMIN: "roles/resourcemanager.projectIamAdmin",
  ORGANIZATION_ADMIN: "roles/resourcemanager.organizationAdmin",
}
////////////////////
// Office 365
///////////////////

export const DIRECTORY_OBJECTS = {
  ODATA: "@odata.id"
}

//Licenses
export const MICROSOFT_LICENSE_ID = {
  BUSINESS_BASIC: "O365_BUSINESS_ESSENTIALS",
  BUSINESS_STANDARD: "O365_BUSINESS_PREMIUM",
  BUSINESS_PREMIUM: "SPB",
}
export const MICROSOFT_LICENSE_GUID = {
  BUSINESS_BASIC: "3b555118-da6a-4418-894f-7df1e2096870",
  BUSINESS_STANDARD: "f245ecc8-75af-4f8e-b61f-27d8114de5f3",
  BUSINESS_PREMIUM: "cbdc14ab-d96c-4c30-b9f4-6ada7cdc1d46",
}

// // STRIPE PRICES
// export const MONTHLY_SUBSCRIPTION = "price_1JbvqlAtcfjD5l2xsC391aNP"
// export const YEARLY_SUBSCRIPTION = "price_1JcS94AtcfjD5l2xQR2OASWf"
// export const TEST_DAILY = "price_1JhQCYAtcfjD5l2xULrc3Z4R"
// export const TEST_YEARLY = "price_1JhQD9AtcfjD5l2xA583vpGw"


// export const SUBSCRIPTION_TYPE_FREE_TRIAL = "FREE-TRIAL"
// export const SUBSCRIPTION_TYPE_FREE = "FREE"
// export const SUBSCRIPTION_TYPE_MONTHLY = "MONTHLY"
// export const SUBSCRIPTION_TYPE_YEARLY = "YEARLY"

export const BILLING_TYPE_MONTHLY = "MONTH"
export const BILLING_TYPE_ANNUALY = "YEAR"
export const BILLING_TYPE_TRIALING = "TRIALING"
export const BILLING_TYPE_PAST_DUE = "PAST_DUE"
export const BILLING_TYPE_ACTIVE = "ACTIVE"
export const BILLING_TYPE_CANCELED = "CANCELED"
export const BILLING_TYPE_UNPAID = "UNPAID"
// SubscriptionStatus = "active"
// SubscriptionStatus = "all"
// SubscriptionStatus = "canceled"
// SubscriptionStatus = "incomplete"
// SubscriptionStatus = "incomplete_expired"
// SubscriptionStatus = "past_due"
// SubscriptionStatus = "trialing"
// SubscriptionStatus = "unpaid"



export const BILLING_DISCOUNT_CHECK = true

export const BOOLEAN_TRUE = "TRUE"
export const BOOLEAN_FALSE = "FALSE"

export const INPUT_CHARACTER_LIMIT = {
  NAME: 255,
  DESCRIPTION: 500,
  ZOOM_ROLE: 100,
  JOB_TITLE: 255,
}


// SALESFORCE ALLOWED USER LICENSES
// These are the names of user licenses allowed in react-select 
// when creating/activating a Salesforce Account
// comment out licenses not applicable for Chatter and Permission Set
export const SALESFORCE_USER_LICENSES: string[] = [
  // "XOrg Proxy User",
  // "Force.com - Free",
  // "Force.com - App Subscription",
  // "Work.com Only",
  "Identity",
  "Chatter Free",
  "Salesforce",
  "Salesforce Platform"
  // "Chatter External" - users with this user License can't be added directly to a Chatter group. It can only be added thru invite in the Salesforce console.
]

export const LOADING_STATES_MODULE = {
  GROUP: "group",
  IMPORT_GOOGLE_USERS: "import_google_users",
  CONNECTED_GROUPS: "connected-groups",
  GROUP_LIST: "group-list",
  GROUP_MEMBERS: "group-members",
  GROUP_CARD: "group-card",
  COMPANY_CARD: "company-card",
  DEPARTMENT_LIST: "department-list",
  GROUP_INFINITE_SCROLL: "group-infinite-scroll",
  USER_INFINITE_SCROLL: "user-infinite-scroll",
  DELETED_USER_INFINITE_SCROLL: "deleted-user-infinite-scroll",
  ZOOM_REGISTRANTS: "zoom-registrants",
  IMPORT_GOOGLE_GROUPS: "import_google_groups",
  COMPANIES: "companies",
  GROUP_USERS: "group-users",
  CONNECTED_ACCOUNTS: "connected-accounts",
  THIRD_PARTY_ACCESS: "third-party-access",
  ACTION_ITEMS: "action-items",
  AWS_ACCESS_KEYS: "aws-access-keys",
  MANAGE_ACCESS_MODAL: "manage-access-modal"
}

export const YOU_SEARCH = "you";
export const JIRA_PROJECT_FIELDS = {
  NAME: {
    maxLength: 50,
  },
  DESCRIPTION: {
    maxLength: 150,
  },
  KEY: {
    maxLength: 10,
    pattern: /^[^ !"`'#%&,:;<>=@{}~_\$\(\)\*\+\/\\\?\[\]\^\|\.\-]+$/,
    patternValidationMessage: "Project keys must start with an uppercase letter, followed by one or more uppercase alphanumeric characters. Special characters are not allowed."
  },
  trimValidationMessage: (value: string) => `${value} cannot include leading and trailing spaces.`,
  maxLengthValidationMessage: (value: number) => `Must be ${value} characters or less.`,
}

export const ROLE_MANAGEMENT_PERMISSIONS = [
  "ADD_ROLE",
  "EDIT_ROLE",
  "REMOVE_ROLE",
  "ASSIGN_ROLE",
  "UNASSIGN_ROLE"
];

export const GROUP_MANAGEMENT_PERMISSIONS = [
  "ADD_GROUP",
  "EDIT_GROUP",
  "CLONE_GROUP",
  "BRANCH_GROUP",
  "REMOVE_GROUP_INTEGRATION",
  "ADD_GROUP_INTEGRATION",
  "ADD_GROUP_MEMBER",
  "REMOVE_GROUP_MEMBER",
  "REMOVE_GROUP",
  "MERGE_GROUP"
]
export const DEPARTMENT_MANAGEMENT_PERMISSIONS = [
  "ADD_DEPARTMENT",
  "EDIT_DEPARTMENT",
  "REMOVE_DEPARTMENT",
]
export const ADD_MEMBERS_TAB = {
  GROUPS: "GROUPS_TAB",
  INDIVIDUAL: "INDIVIDUALS_TAB"
}

export const DROP_REJECTED = {
  SIZE: "file-too-large",
  TYPE: "file-invalid-type",
}

export const ORIGIN = {
  DEFAULT: "SaaSConsole",
  GOOGLE: "Google",
  MICROSOFT: "Microsoft",
}

export const ACTION_DISPLAY = {
  ADD: "added to your",
  DELETE: "deleted from your",
  REMOVE: "removed from your",
}

export const ACTION_TITLE_DISPLAY = {
  ADD: "Adding Members",
  DELETE: "Deleting Members",
  REMOVE: "Removing Members",
}

export const ACTION_MEMBERS = {
  ADDED: "added",
  REMOVED: "removed",
}

export const RESEND_EMAIL_TIMER = 60;

export const RESEND_USER_ZOOM = 360;

export const IntegrationsHaveMultiOptions = [INTEG_SLUG.GOOGLE_CLOUD.GCP, INTEG_SLUG.GOOGLE_CLOUD.FIREBASE, INTEG_SLUG.JIRA.PROJECTS]

export const LOG_EVENT= {
  GOOGLE: {
    INVITE_USER: "GOOGLE:INVITE_USER",
    UPDATE_MEMBER_PERMISSIONS: "GOOGLE:ADMIN_UPDATE_PERMISSIONS",
    REMOVE_MEMBER: "GOOGLE:ADMIN_REMOVE_MEMBER",
    
  }
}

export const SOURCE_TYPE = {
  SYSTEM: "System",
  INTEGRATION: "Integration"
}

export const NOTIFICATION_TYPE = {
  REQUEST_CONNECT_INTEGRATION: "REQUEST_CONNECT_INTEGRATION",
  REQUEST_DISCONNECT_INTEGRATION: "REQUEST_DISCONNECT_INTEGRATION",
  REQUEST_REMOVE_INTEGRATION: "REQUEST_REMOVE_INTEGRATION"
}

//* CRON JOB TYPES
export const JOB_TYPE = {
  ADD_GROUP_MEMBERS: "JOB_ADD_GROUP_MEMBERS",
  REMOVE_GROUP_MEMBERS: "JOB_REMOVE_GROUP_MEMBERS",
  MOVE_GROUP_MEMBERS: "JOB_MOVE_GROUP_MEMBERS",
  OFF_BOARDING_USER: "JOB_OFF_BOARDING_USER",
  ON_BOARDING_USER: "JOB_ON_BOARDING_USER"
}

export interface ISelect {
    value: string
    label: string
    members?: any
    userRole?: string[]
}

export interface ApplicationsProps {
  groupId: string
  groupName?: string
  groupMembers?
  integration
  groupIntegrations
  onDone: () => void
  currentUser
}

export interface IJiraConnectModal {
  show:           boolean
  handleClose:    () => void
  companyId:      string
  integrationId:  string
  activeUser:any
}

export interface IJiraUserProps {
    connectedItems?: {
      label:  string
      value:  string
    }[]
    data: {
      label:  string 
      value:  string
    }[]
    application: {
      ConnectedDrives:          string[]
      ConnectedItems:           string[]
      CreatedAt:                string
      DisplayPhoto:             string
      IntegrationAccessToken:{
        access_token:           string
        expiry:                 string
      }
      IntegrationDescription:   string
      IntegrationID:            string
      IntegrationName:          string
      IntegrationSlug:          string
      PK:                       string
      SK:                       string
    }
    JiraUsersOnChange:  (input) => void
    JiraFetch:          () => void
    isLoading:          boolean
  }

export interface IPermission {
    PK                      : string
    SK                      : string
    PermissionCategoryCode  : string
    PermissionCategoryName  : string
    PermissionCode          : string
    PermissionName          : string
    SearchKey               : string
    TYPE                    : string
}

export const InitialPermission: IPermission = {
  PK                      : '',
  SK                      : '',
  PermissionCategoryCode  : '',
  PermissionCategoryName  : '',
  PermissionCode          : '',
  PermissionName          : '',
  SearchKey               : '',
  TYPE                    : '',
}

export interface IRole {
  PK              : string
  SK              : string
  RoleID          : string
  RoleName        : string
  CompanyID       : string
  RolePermissions : string[]
  SearchKey       : string
  CreatedAt       : string
  CreatedBy       : string
  Total           : number
  TYPE            : string
  Users           : any[]
}

export const InitialRole = {
  PK              : '',
  SK              : '',
  RoleID          : '',
  RoleName        : '',
  CompanyID       : '',
  RolePermissions : [],
  SearchKey       : '',
  CreatedAt       : '',
  CreatedBy       : '',
  Total           : 0,
  TYPE            : '',
  Users           : []
}
import { IPayload } from "../../interfaces/payloads/payload"




export type AssociatedAccountsType = {
    [key: string]: string[];
  };

export type OracleDBAssociatedAccount = {
    databaseId: string,
    databaseName: string,
    username: string,
    userId: string
}
export type UsersType = {
    PK: string
    SK: string
    UserID: string
    FirstName: string
    LastName: string
    DisplayPhoto: string
    Email: string
    Status: string
    Roles?: any //UNKNOWN TYPE
    // Permissions?: any //UNKNOWN TYPE
    UserRole: string[]
    UserType: string
    CreatedAt: string
    DeletedAt?: number
    Groups?: GroupType[]
    Origin?: string
    AssociatedAccounts?: AssociatedAccountsType[]
    OracleDBAssociatedAccounts?: OracleDBAssociatedAccount[]
}[]

export type UserType = {
    PK: string
    SK: string
    UserID: string
    FirstName: string
    LastName: string
    DisplayPhoto: string
    Email: string
    Status: string
    Roles?: any //UNKNOWN TYPE
    // Permissions?: any //UNKNOWN TYPE
    UserRole: string[]
    CreatedAt: string
    DeletedAt?: number
    Groups?: GroupType[]
    Origin?: string
    AssociatedAccounts?: AssociatedAccountsType[]
}

export type GroupType = {
    PK: string
    SK: string
    GroupID: string
    CompanyID: string
    DepartmentID: string
    GroupName: string
    GroupColor: string
    Status: string
    Type: string
    NewGroup: string
    CreatedAt: string
    UpdatedAt: string
    SearchKey: string
}




//?REDUX
export interface AddUserSocketResponse {
    message: string
    status: {
        code: string
        message: string
    }
    statusCode: number
    data: {
        itemCount: number
        doneCount: number
    }
}


//?ADD USER WEBSOCKET
export type AddUsersWebSocketRequestPayload = {
    users: {
        FirstName: string
        LastName: string
        Email: string
        Origin: string
    }[],
    enable_login: boolean
    is_company_admin: boolean
} & IPayload

export type AddUserWebSocketSuccessPayload = {
    type: "CONNECTED" | "SUCCESS" | "DISCONNECTED"
    response: AddUserSocketResponse
}

//?REMOVE USER WEBSOCKET
export interface RemoveUserSocketResponse {
    message: string
    status: {
        code: string
        message: string
    }
    statusCode: number
    data: {
        users: string[]
        itemCount: number
        doneCount: number
    }
}

export type RemoveUsersWebSocketRequestPayload = {
    users: string[],
    remove_integration_accounts: boolean
} & IPayload

export type RemoveUserWebSocketSuccessPayload = {
    type: "CONNECTED" | "DISCONNECTED" | "SUCCESS" | "ERROR"
    response: RemoveUserSocketResponse
}

//?REMOVE USER INTEGRATION WEBSOCKET
export interface RemoveUserIntegrationSocketResponse {
    message: string
    status: {
        code: string
        message: string
    }
    statusCode: number
    data: {
        result: string[]
    }
}

export type RemoveUserIntegrationWebSocketRequestPayload = {
    users: string[],
} & IPayload

export type RemoveUserIntegrationWebSocketSuccessPayload = {
    type: "CONNECTED" | "DISCONNECTED" | "SUCCESS" | "ERROR"
    response: RemoveUserIntegrationSocketResponse
}

//?RESTORE USER WEBSOCKET
export type RestoreUserSocketRequestPayload = {
    userIds: string[]
} & IPayload

export type RestoreUserWebSocketSuccessPayload = {
    type: "CONNECTED" | "DISCONNECTED" | "SUCCESS"
    response: RestoreUserSocketResponse
}
export interface RestoreUserSocketResponse {
    message: string
    status: {
        code: string
        message: string
    }
    statusCode: number
    data: {
        users: string[]
        itemCount: number
        doneCount: number
    }
}

//?DELETE USER WEBSOCKET
export type DeleteUserSocketRequestPayload = {
    userIds: string[]
} & IPayload

export type DeleteUserWebSocketSuccessPayload = {
    type: "CONNECTED" | "DISCONNECTED" | "SUCCESS"
    response: DeleteUserSocketResponse

}
export interface DeleteUserSocketResponse {
    message: string
    status: {
        code: string
        message: string
    }
    statusCode: number
    data: {
        users: string[]
        itemCount: number
        doneCount: number
    }
}

export interface NormalUserAccountCreationRequestPayload {
    email: string
    integrationID: string
}

export interface NormalUserMatchAccountRequestPayload {
    requesterEmail: string
    toMatchEmail: string
    integrationID: string
}
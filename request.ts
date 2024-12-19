import axios from "axios";
import { ICompanySettings } from "../interfaces/modules/Settings";
import { END_POINTS } from "./api";
import axiosInstance from "./axios";
import { GROUP_MEMBER } from "./gcloud/directory/interface";

import { SubsGetSubscriptionMemberResponse, SubsGetSubscriptionResponse } from "../interface-new/Subscription";
import * as companyTypes from "../interfaces/services/company";
import { IGroup, IGroupCopyIntegration } from "../interfaces/modules/Groups";
import { NormalUserAccountCreationRequestPayload, NormalUserMatchAccountRequestPayload } from "../redux/users/interface";
import { AuthTokenLoginPayload } from "../redux/auth/types";


export const multipleRequest: any = request => axios.all(request);

// Authentication
export const authRequests = {
    signIn: (body) => axiosInstance.post(END_POINTS.SIGN_IN, body),
    signOut: (user) => axiosInstance.put(END_POINTS.SIGN_OUT, user),
    resendActivation: (query) => axiosInstance.put(`${END_POINTS.RESEND_ACTIVATION}?${query}`),
    // activateUser: (token) => axiosInstance.put(`${END_POINTS.ACTIVATE_USER}?token=${token}`),
    activateUser: (params) => axiosInstance.put(END_POINTS.ACTIVATE_USER, {}, { params }),
    signUp: (body, params) => axiosInstance.post(END_POINTS.SIGN_UP, body, { params }),
    requestPasswordReset: (email) => axiosInstance.post(END_POINTS.REQUEST_PASSWORD_RESET, email),
    resetPassword: (token, newPassword) => axiosInstance.post(END_POINTS.RESET_PASSWORD + `?token=${token}`, newPassword),
    changePassword: (body) => axiosInstance.put(END_POINTS.CHANGE_PASSWORD, body),
    checkUserPassword: (body) => axiosInstance.post(END_POINTS.CHECK_PASSWORD, body),
    resendEmailResetPassword: (body) => axiosInstance.post(END_POINTS.RESEND_EMAIL_RESET_PASSWORD, body),
    checkTokenValidity: (token) => axiosInstance.get(END_POINTS.CHECK_TOKEN_VALIDITY + `?token=${token}`),
    getGoogleAuth: () => axiosInstance.get(`${END_POINTS.SIGN_IN}/${END_POINTS.GOOGLE}`),
    googleSignIn: (code: string) => axiosInstance.post(`${END_POINTS.SIGN_IN}/${END_POINTS.GOOGLE}`, {}, {params: {code}}),
    getMsOfficeAuth: () => axiosInstance.get(`${END_POINTS.SIGN_IN}/${END_POINTS.MICROSOFT}`),
    microsoftSignIn: (code: string) => axiosInstance.post(`${END_POINTS.SIGN_IN}/${END_POINTS.MICROSOFT}`, {}, {params: {code}}),
    migrateAccount: (body) => axiosInstance.post(`${END_POINTS.MIGRATE}`, body, {}),
    authTokenLogin: (params: AuthTokenLoginPayload) => axiosInstance.post(`${END_POINTS.TOKEN_LOGIN}` ,{}, {params}),
};

// People
export const userRequests = {
    getAllUsers: () => axiosInstance.get(END_POINTS.GET_ALL_USERS),
    getUsers: (params) => axiosInstance.get(END_POINTS.USERS, { params: params }),
    getUser: (userID: string, params) => axiosInstance.get(`${END_POINTS.USERS}/${userID}`, { params }),
    getSignedInUser: () => axiosInstance.get(`${END_POINTS.USERS}/me`),
    addUsers: (body) => axiosInstance.post(END_POINTS.USERS, body),
    updateUser: (userID: string, body) => axiosInstance.put(`${END_POINTS.USERS}/${userID}`, body),
    updateUserStatus: (body) => axiosInstance.patch(`${END_POINTS.USERS}/status/`, body),
    updateUserEmail: (userID: string, details) => axiosInstance.patch(`${END_POINTS.USERS}/email/${userID}`, { details }),
    cloneUserGroups: (userID: string, body) => axiosInstance.post(`${END_POINTS.USERS}/groups/clone/${userID}`, body),
    getUserCompanies: (userID: string, params) => axiosInstance.get(`${END_POINTS.GET_USER_COMPANIES}/${userID}`, { params }),
    addToGroup: (body) => axiosInstance.post(END_POINTS.ADD_GROUP_MEMBER_USERS, body),
    inviteCompanyAdmin: (body) => axiosInstance.post(`/companies/invite`, body),

    saveGoogleAccessToken: (body) => axiosInstance.put(`${END_POINTS.USERS}/token/google`, body),
    // saveJiraKeys: (body) => axiosInstance.put(`${END_POINTS.JIRA_ADD_KEY}`, body),
    removeJiraKeys: () => axiosInstance.put(`${END_POINTS.JIRA_REMOVE_KEY}`),

    // Mailing Newly Created Users for Integrations
    integrationNewUserMail: (body) => axiosInstance.post(END_POINTS.INTEG_NEW_USER_MAIL, body),

    skipSetupWizard: (companyId) => axiosInstance.post(`${END_POINTS.SKIP_SETUP_WIZARD}/${companyId}`, {}),
    finishTour: (body) => axiosInstance.patch(`${END_POINTS.FINISH_TOUR}`, body),
    confirmVerification: (body) => axiosInstance.patch(END_POINTS.CONFIRM_VERIFICATION, body),
    updateUserProfile: (userID: string, body) => axiosInstance.put(`${END_POINTS.UPDATE_PROFILE}/${userID}`, body),
    updateIntegrationAccountProfile: (userID: string, body) => axiosInstance.put(`${END_POINTS.UPDATE_ACCOUNT}/${userID}`, body),
    updateIntegrationAccounts: (userID: string, body) => axiosInstance.post(`${END_POINTS.UPDATE_ACCOUNTS}/${userID}`, body),
    updateUserInfo: (userID: string, body) => axiosInstance.put(`${END_POINTS.UPDATE_USER_INFO}/${userID}`, body),
    updateUserPhoto: (userID: string, body) => axiosInstance.patch(`${END_POINTS.UPDATE_USER_PHOTO}/${userID}`, body),

    removeGroupMembersToGroupIntegrations: (userID: string, params) => axios.delete(`${END_POINTS.USERS}/${userID}`, { params }),
    deleteUser: (userID: string, params) => axiosInstance.delete(`${END_POINTS.USERS}/${userID}`, { params }),
    removeMultipleUsers: (params) => axiosInstance.delete(`${END_POINTS.REMOVE_MULTIPLE_USERS}`, { params }),
    restoreUsers: (body) => axiosInstance.post(`${END_POINTS.USERS}/restore`, body),
    permanentlyDeleteUsers: (body) => axiosInstance.delete(`${END_POINTS.USERS}/permanently-delete`, { data: body }),

    getActiveConnectedGroups: (params) => axiosInstance.get<IGroup[]>(`${END_POINTS.USERS}/groups/count`, { params }),
    manageAccess:(body) => axiosInstance.post(`${END_POINTS.MANAGE_ACCESS}`, body),
    requestToJoinGroup: (body) => axiosInstance.post(END_POINTS.REQUEST_TO_JOIN_GROUP, body),
    normalUserAccountCreationRequest: (body: NormalUserAccountCreationRequestPayload) => axiosInstance.post(`${END_POINTS.MANAGE_ACCESS}${END_POINTS.REQUEST_ACCOUNT_CREATION}`, body),
    normalUserMatchAccountRequest: (body: NormalUserMatchAccountRequestPayload) => axiosInstance.post(`${END_POINTS.MANAGE_ACCESS}${END_POINTS.REQUEST_MATCH_ACCOUNT}`, body),
    normalUserInviteRequest: (body: NormalUserAccountCreationRequestPayload) => axiosInstance.post(`${END_POINTS.MANAGE_ACCESS}${END_POINTS.REQUEST_INVITE_ACCOUNT}`, body),
    createJob:(body) => axiosInstance.post(`${END_POINTS.USERS}/job`, body),
    acceptRequest: (body, id) => axiosInstance.post(`${END_POINTS.REQUESTS}/${id}/accept`, body),
    rejectRequest: (body, id) => axiosInstance.post(`${END_POINTS.REQUESTS}/${id}/reject`, body),
    createUserNew: (body) => axiosInstance.post(END_POINTS.CREATE_USER_NEW, body),
    createCronJobUser: (body) => axiosInstance.post(`users/cron_job`, body),
    checkEmailDuplicate: (body) => axiosInstance.post(`users/check-email`, body),
}

// Companies
export const companyRequests = {
    getAllCompanies: () => axiosInstance.get(END_POINTS.GET_ALL_COMPANIES),
    getCompaniesByUser: (userID: string, params) => axiosInstance.get(`${END_POINTS.GET_COMPANIES_BY_USER}/${userID}`, { params }),
    addBookmark: (company) => axiosInstance.post(END_POINTS.ADD_BOOKMARK_COMPANY, company),
    deleteBookmark: (company) => axiosInstance.post(END_POINTS.DELETE_BOOKMARK_COMPANY, company),
    getCompany: (companyID: string, params) => axiosInstance.get(`${END_POINTS.COMPANIES}/${companyID}`, { params }),
    updateCompany: (companyID: string, details) => axiosInstance.put(`${END_POINTS.COMPANIES}/${companyID}`, details),
    updateCompanyInfo: (companyID: string, details) => axiosInstance.put(`${END_POINTS.COMPANIES}/${companyID}`, details),
    updateCompanyLogo: (companyID: string, details) => axiosInstance.patch(`${END_POINTS.UPDATE_COMPANY_LOGO}/${companyID}`, details),
    addCompany: (company) => axiosInstance.post(END_POINTS.COMPANIES, company),
    changeActiveCompany: (userID: string, body) => axiosInstance.patch(`${END_POINTS.ACTIVE_COMPANY}/${userID}`, body),
    getActiveCompany: (userID: string) => axiosInstance.get(`${END_POINTS.ACTIVE_COMPANY}/${userID}`),
    createCompanyWithLinks: (body: FormData) => axiosInstance.post<companyTypes.CreateCompanyWithLinkResponse>(`/companies/create`, body),
    getCompanyUsersCount: (companyId: string) => axiosInstance.get(`${END_POINTS.COMPANIES}/${companyId}/users/count`),
    getCompanyCount: (userID: string) => axiosInstance.get(`${END_POINTS.COMPANIES}/users/${userID}/count`),
    createCompanySSO: (company) => axiosInstance.post(`${END_POINTS.COMPANIES}/${END_POINTS.SSO}`, company),

    getCompanyUsers: (companyId: string) => axiosInstance.get(`${END_POINTS.COMPANIES}/${companyId}/users`),
    getCompanyGroups: (companyId: string) => axiosInstance.get(`${END_POINTS.COMPANIES}/${companyId}/groups`),

    enableUserAccess: (companyId: string) => axiosInstance.post(`${END_POINTS.COMPANIES}/${companyId}/settings/enable-user-access`),
}

// Groups
export const groupRequests = {
    getGroup: (body) => axiosInstance.get(`${END_POINTS.GROUP}?group_id=${body.group_id} `),
    getGroupApplications: (groupId: string) => axiosInstance.get(`${END_POINTS.GROUPS}/${groupId}/applications`),
    getAllGroups: () => axiosInstance.get(END_POINTS.GET_ALL_GROUPS),
    getGroupDepartment: (groupID: string) => axiosInstance.get(`${END_POINTS.GROUPS}/${groupID}/department`),
    // getAllGroupsByCompanyId: (body) => axiosInstance.get(`${ END_POINTS.GET_ALL_GROUPS_BY_COMPANYID }?company_id = ${ body.company_id }& status=${ body.status }& department_id=${ body.department_id }& limit=${ body.limit }& last_evaluated_key=${ body.last_evaluated_key }& include=members, integrations, sub - integrations & key=${ body.key } `),
    getAllGroupsByCompanyId: (params) => axiosInstance.get(END_POINTS.GET_ALL_GROUPS_BY_COMPANYID, { params }),
    getByDepartments: (params) => axiosInstance.get(`${END_POINTS.GROUPS} `, { params }),
    addGroup: (group) => axiosInstance.post(END_POINTS.GROUPS, group),
    addMember: (group) => axiosInstance.post(END_POINTS.ADD_MEMBER, group),
    addBookmark: (group) => axiosInstance.post(END_POINTS.ADD_BOOKMARK_GROUP, group),
    deleteBookmark: (group) => axiosInstance.post(END_POINTS.DELETE_BOOKMARK_GROUP, group),
    addIntegration: (group) => axiosInstance.post(END_POINTS.ADD_GROUP_INTEGRATION, group),
    deleteIntegration: (group) => axiosInstance.post(END_POINTS.DELETE_GROUP_INTEGRATION, group),
    deleteGroup: (group) => axiosInstance.post(END_POINTS.DELETE_GROUP, group),
    updateMember: (group) => axiosInstance.post(END_POINTS.UPDATE_MEMBER, group),
    updateGroup: (groupID: string, group) => axiosInstance.put(`${END_POINTS.GROUPS}/${groupID}`, group),
    cloneGroup: (groupID: string, group) => axiosInstance.post(`${END_POINTS.CLONE_GROUP}/${groupID}`, group),
    branchGroup: (groupID: string, group) => axiosInstance.post(`${END_POINTS.BRANCH_GROUP}/${groupID}`, group),
    mergeGroup: (group) => axiosInstance.post(END_POINTS.MERGE_GROUP, group),
    getAllGroupsCount: (body) => axiosInstance.get(`${END_POINTS.GET_ALL_GROUPS_COUNT}`, body),
    getGroupMembersCount: (params) => axiosInstance.get(END_POINTS.GET_GROUP_MEMBERS_COUNT, {params}),

    getGroupMembers: (groupID: string, includeSubGroup?: boolean) => axiosInstance.get(`${END_POINTS.GROUPS}/${groupID}/members?includeSubGroup=${includeSubGroup}`),

    //connect  integration to gorup
    connectIntegration: (groupId: string, body) => axiosInstance.put(`/groups/${groupId}/connect_integration`, body),
    connectIntegrationToGroups: (groups) => axiosInstance.post(END_POINTS.CONNECT_INTEGRATION_TO_GROUPS, groups),

    updateGroupSubIntegration: (params, body) => axiosInstance.put(`${END_POINTS.GROUPS}/sub_integrations?${params}`, body),
    // Group Members
    addMemberUsers: (members) => axiosInstance.post(END_POINTS.ADD_GROUP_MEMBER_USERS, { members }),
    addMemberGroups: (body) => axiosInstance.post(END_POINTS.ADD_GROUP_MEMBER_GROUPS, body),
    deleteMembers: (body) => axiosInstance.post(END_POINTS.DELETE_MEMBER, body),
    deleteMembersGroup: (groupID: string, members) => (axiosInstance.delete(`${END_POINTS.DELETE_MEMBER_GROUP}${groupID}/group_members`, { data: members })),

    suggestedGroups: (groupId: string, params) => axiosInstance.get(`${END_POINTS.SUGGESTED_GROUPS}/${groupId}`, { params }),
    addUsersToGroups: (body) => axiosInstance.post(END_POINTS.ADD_USERS_TO_GROUPS, body),

    // Mailing Newly Created Users for INtegrations
    integrationNewUserMail: (body) => axiosInstance.post(END_POINTS.INTEG_NEW_USER_MAIL, body),

    // import google admin groups
    importGroups: (body, params) => axiosInstance.post(END_POINTS.GA_IMPORT_GROUPS, body, { params }),
    groupsImport: (body, params) => axiosInstance.post(END_POINTS.GA_GROUPS_IMPORT, body, { params }),
    
    getGroupCountByCompany: (companyID: string) => axiosInstance.get(`${END_POINTS.GROUPS}/companies/${companyID}/count`),

    copyGroupsIntegration: (groupID: string, body) => axiosInstance.post(`${END_POINTS.GROUPS}/copy/${groupID}`, body, {  }),

    //for notification request
    removeIntegrationRequest: (body) => axiosInstance.post(`${END_POINTS.DELETE_GROUP_INTEGRATION_REQUEST}`, body),

    //* cron job group members
    saveCronJobGroupMembers: (body) => axiosInstance.post(END_POINTS.CRON_JOB_GROUP_MEMBERS, body),
    //* get group integrations
    getGroupIntegrations: (groupID: string) => axiosInstance.get(`${END_POINTS.GROUP_INTEGRATIONS}/${groupID}`),
    //* move members to another groups
    moveMembersToAnotherGroups: (body) => axiosInstance.post(END_POINTS.MOVE_MEMBERS_TO_ANOTHER_GROUPS, body),

    importExternalGoogleGroup: (body) => axiosInstance.post(`${END_POINTS.GROUPS}/import/external/google`, body),
}

// Roles
export const roleRequest = {
    getRole: (params: string) => axiosInstance.get(`${END_POINTS.GET_ROLE}?roleId=${params}`),
    numOfUsers: (body) => axiosInstance.get(`${END_POINTS.GET_NUM_USERS}?roleId=${body}`),
    getAllRoles: (company_id: string, key: string = "") => axiosInstance.get(`${END_POINTS.GET_ALL_ROLES}?company_id=${company_id}&key=${key}`),
    addNewRole: (body) => axiosInstance.post(END_POINTS.ADD_NEW_ROLE, body, { headers: { "Content-Type": "multipart/form-data", "Accept": "*/*" } }),
    assignPermissions: (body) => axiosInstance.put(END_POINTS.ASSIGN_PERMISSIONS, body, { headers: { "Content-Type": "multipart/form-data", "Accept": "*/*" } }),
    assignRole: (body) => axiosInstance.post(END_POINTS.ASSIGN_ROLE, body),
    unassignRole: (body) => axiosInstance.post(END_POINTS.UNASSIGN_ROLE, body),
    deleteRole: (body: FormData) => axiosInstance.post(`${END_POINTS.DELETE_ROLE}`, body),
    requestRoles :(body) => axiosInstance.post(END_POINTS.REQUEST_ROLES, body),
    
}

// Permissions
export const permissionsRequest = {
    getAllPermissions: () => axiosInstance.get(END_POINTS.GET_ALL_PERMISSIONS),
}

// Logs
export const logRequests = {
    getAllLogs: () => axiosInstance.get(END_POINTS.GET_ALL_LOGS),
    getLogs: (params) => axiosInstance.get(END_POINTS.LOGS, { params }),
    getLog: (logID: string) => axiosInstance.get(`${END_POINTS.LOGS}/${logID}`),
    deleteLogs: () => axiosInstance.patch(END_POINTS.LOGS),
    getNewLogs: (params) => axiosInstance.get(END_POINTS.NEW_LOGS),
}



// Integrations
export const integrationsRequests = {
    getIntegrations: (params) => axiosInstance.get(END_POINTS.INTEGRATIONS, { params }),
    connectIntegration: (body) => axiosInstance.post(END_POINTS.CONNECT_INTEGRATION, body),
    disconnectIntegration: (body) => axiosInstance.post(END_POINTS.DISCONNECT_INTEGRATION, body),
    requestIntegration: (body) => axiosInstance.post(`${END_POINTS.INTEGRATIONS}/request`, body),
    addIntegration: (body) => axiosInstance.post(`${END_POINTS.INTEGRATIONS}/add`, body),

    getIntegrationUIDs: (integrationId: string, params?) => axiosInstance.get(`${END_POINTS.INTEGRATION_UIDS}/${integrationId}`, { params }),
    sendMapUIDEmails: (integrationId: string, body) => axiosInstance.post(`${END_POINTS.INTEGRATION_UIDS}/${integrationId}/send-emails`, body),
    saveIntegrationUID: (params, body) => axiosInstance.post(`${END_POINTS.INTEGRATION_UIDS}`, body, { params }),
    mapIntegrationUID: (integrationId: string, body) => axiosInstance.post(`${END_POINTS.BATCH_INTEGRATION_UIDS}/${integrationId}`, body),
    changeIntegrationUIDStatus: (integrationId: string, body) => axiosInstance.patch(`${END_POINTS.INTEGRATION_UIDS}/${integrationId}`, body),

    listConnectedGroupsByIntegration: (integrationId: string) => axiosInstance.get(`${END_POINTS.INTEGRATIONS}/connected/groups/${integrationId}`),
    getIntegrationProfile: (integrationId: string, params?) => axiosInstance.get(`${END_POINTS.INTEGRATIONS}/${integrationId}`, { params }),

    addSubIntegration: (body) => axiosInstance.post(`${END_POINTS.SUB_INTEGRATIONS}/add`, body),

    getIntegrationCountByCompany: (companyID: string) => axiosInstance.get(`${END_POINTS.INTEGRATIONS}/companies/${companyID}/count`),

    getDoneIntegrationGuides: (userID: string) => axiosInstance.get(`${END_POINTS.INTEGRATIONS}/guide-done/${userID}`),
    updateIsIntegrationGuideDone: (integrationSlug: string) => axiosInstance.put(`${END_POINTS.INTEGRATIONS}/guide-done/${integrationSlug}`),

    getSortedGroupsByPopularity: (integrationID: string, params) => axiosInstance.get(`${END_POINTS.INTEGRATIONS}/${integrationID}/groups`, { params }),

    migrateThirdPartyApp: (body) => axiosInstance.post(END_POINTS.MIGRATE_INTEGRATIONS, body),
    sendEmailIntegrationCredentials: (body) => axiosInstance.post(END_POINTS.SEND_EMAIL_INTEGRATION_CREDENTIALS, body),
    allowExternalUsers: (body) => axiosInstance.post(END_POINTS.ALLOW_EXTERNAL_USERS, body),

    requestConnectDisconnectIntegration: (body) => axiosInstance.post(`${END_POINTS.INTEGRATIONS}/request-integrations-action`, body),
    getIntegrationBySlug: (integrationSlug: string) => axiosInstance.get(`${END_POINTS.INTEGRATIONS}/slug/${integrationSlug}`),
}

// Notifications
export const notificationsRequests = {
    getCurrentUserNotifications: () => axiosInstance.get(END_POINTS.MY_NOTIFICATIONS),
    seenNotification: (notificationId: string) => axiosInstance.patch(`${END_POINTS.SEEN_NOTIFICATION}/${notificationId}`),
    deleteNotification: (notificationId: string) => axiosInstance.delete(`${END_POINTS.NOTIFICATIONS}/${notificationId}`),
    markAllAsRead: () => axiosInstance.put(`${END_POINTS.MY_NOTIFICATIONS}/read_all`),
    getCurrentUserNotificationsPaginated: (lastEvaluatedKey: null | String = null, all = false, sortorder: String = 'descending', pagelimit: number = 5) => axiosInstance.get(
        lastEvaluatedKey ? `/notifications/pagination?lastEvaluatedKey=${lastEvaluatedKey}&sortOrder=${sortorder}&pageLimit=${pagelimit}` : `/notifications/pagination?all=${all}&sortOrder=${sortorder}`
    )
}

// Subscriptions
export const subscriptionRequests = {
    getCompanySubscription: () => axiosInstance.get<SubsGetSubscriptionResponse>(`/subscription`),
    getSubscriptionMembers: () => axiosInstance.get<SubsGetSubscriptionMemberResponse[]>(`/subscription/members`),
    // getCompanySubscriptions: () => axiosInstance.get<subscriptionTypes.Subscriptions>(`${END_POINTS.SUBSCRIPTIONS}`),
    // createSubscriptions: (body) => axiosInstance.post(END_POINTS.CREATE_SUBSCRIPTIONS, body),
    // updateSubscriptions: (body) => axiosInstance.post(END_POINTS.UPDATE_SUBSCRIPTIONS, body),
    // getMemberSubscription: (params) => axiosInstance.get(`${END_POINTS.SUBSCRIPTIONS_USERS}`, { params }),
}

// Action Items
export const actionItemsRequests = {
    getActionItems: (params) => axiosInstance.get(END_POINTS.ACTION_ITEMS, { params }),
    deleteActionItem: (actionItemId: string) => axiosInstance.delete(`${END_POINTS.ACTION_ITEMS}/${actionItemId}`),
    createActionItem: (body) => axiosInstance.post(END_POINTS.ACTION_ITEMS, body),
}

// Google Cloud
export const googleCloudRequests = {
    auth: () => axiosInstance.get(END_POINTS.GCLOUD_AUTH),
    tokenFromWeb: (code: string, companyId: string, integrationId: string) => axiosInstance.get(END_POINTS.GCLOUD_TOKEN, {
        params: {
            code,
            company_id: companyId,
            integration_id: integrationId,
        }
    }),
    userInfo: (params) => axiosInstance.get(END_POINTS.GCLOUD_PEOPLE_ME, { params }),
    getUserCustomerInfoByToken: (params) => (axiosInstance.get(`${END_POINTS.GCLOUD_PEOPLE_ME}/tokeninfo`, { params })),
}

export const googleAdminRequests = {
    // MANAGE USERS
    createUser: (body) => axiosInstance.post(END_POINTS.GA_USERS, body),
    getUsers: (params?) => axiosInstance.get(END_POINTS.GA_USERS, { params }),
    getUser: (params, email?: string) => axiosInstance.get(`${END_POINTS.GA_USERS}/${email}`, {params}),
    getUserEmails: (email, params) => axiosInstance.get(`${END_POINTS.GA_USERS}/primary-email/${email}`, { params }),
    deleteGoogleAccount: (email, params) => axiosInstance.delete(`${END_POINTS.GA_USERS}/${email}`, { params }),
    // MANAGE GROUPS
    createGroup: (body) => axiosInstance.post(END_POINTS.GA_GROUPS, body),
    getGroups: (params) => axiosInstance.get(END_POINTS.GA_GROUPS, { params }),
    getGroup: (groupKey: string) => axiosInstance.get(`${END_POINTS.GA_GROUPS}/${groupKey}`),
    // MANAGE GROUP MEMBERS
    getGroupMembers: (groupKey: string, params) => axiosInstance.get(`${END_POINTS.GA_GROUPS}/${groupKey}/members`, { params }),
    SelectAllGroupsMembers: (params) => axiosInstance.get(`${END_POINTS.GA_GROUPS}/all`, { params }),
    getGroupMember: (groupKey: string, memberKey: string) => axiosInstance.get(`${END_POINTS.GA_GROUPS}/${groupKey}/members/${memberKey}`),
    groupHasMember: (groupKey: string, memberKey: string) => axiosInstance.get(`${END_POINTS.GA_GROUPS}/${groupKey}/members/has/${memberKey}`),
    addGroupMembers: (members: GROUP_MEMBER[], groupKey: string) => {
        const requests = members.map(member => {
            const { email } = member;
            return axiosInstance.post(`${END_POINTS.GA_GROUPS}/${groupKey}/members`, {
                email,
                // role,
            })
        })
        return axios.all(requests);
    },
    deleteGroupMembers: (memberKeys: string[], groupKeys: string[]) => {
        const requests = groupKeys.map(groupKey => {
            return memberKeys.map(memberKey => {
                return axiosInstance.delete(`${END_POINTS.GA_GROUPS}/${groupKey}/members/${memberKey}`);
            })
        });
        return axios.all(requests);
    },
    updateGroupMember: (groupKey: string, memberKey: string, member) => axiosInstance.put(`${END_POINTS.GA_GROUPS}/${groupKey}/members/${memberKey}`, member),
    // Recover Google Account
    recoverGoogleAccount: (userKey: string, body, params) => axiosInstance.post(`${END_POINTS.GA_USERS}/${userKey}/restore-account`, body, { params }),
    getCustomerOrgUnits: (userId: string, params) => axiosInstance.get(`${END_POINTS.GA_CUSTOMER}/${userId}/orgunits`, { params }),
    restoreMultipleAccounts: (body, params) => axiosInstance.post(`${END_POINTS.GA_USERS}/restore-multiple-accounts`, body, { params }),
}

export const gcpRequests = {
    // MANAGE PROJECTS
    createProject: (body) => axiosInstance.post(END_POINTS.GCP_PROJECTS, body),
    getProjects: () => axiosInstance.get(END_POINTS.GCP_PROJECTS),
    getProject: (projectKey: string) => axiosInstance.get(`${END_POINTS.GCP_PROJECTS}/${projectKey}`),
    getProjectIamPolicy: (projectKey: string) => axiosInstance.get(`${END_POINTS.GCP_PROJECTS}/${projectKey}/getIamPolicy`),
    setProjectIamPolicy: (projectKey: string, policy) => axiosInstance.post(`${END_POINTS.GCP_PROJECTS}/${projectKey}/setIamPolicy`, { policy }),
    testIamPermissions: () => axiosInstance.post(END_POINTS.GCP_TEST_IAM_PERMISSIONS),
    getUserOrganizations: () => axiosInstance.get(`${END_POINTS.GCP_ORGANIZATIONS}/me`),
    addNewPolicyToOrganization: (body) => axiosInstance.put(`${END_POINTS.GCP_ORGANIZATIONS}/policies`, body),
    getRoles: (pageToken) => axiosInstance.get(`${END_POINTS.GCLOUD_IAM}/roles?pageToken=${pageToken}`),
    getRole: (role: string) => axiosInstance.get(`${END_POINTS.GCLOUD_IAM}/${role}`),
}


//Jira
// export const jiraRequests = {
//     auth: () => axiosInstance.get(JIRA_END_POINTS_DIRECTORY.AUTH),
//     getToken: (code: string) => axiosInstance.get(JIRA_END_POINTS_DIRECTORY.TOKEN, { params: { code } }),
// }



// export const stripeRequests = {
//     getCustomerByUserID: () => axiosInstance.get<stripeTypes.GetCustomerByUserID>(`${END_POINTS.STRIPE_CUSTOMER}`),
//     getStripePaymentMethod: () => axiosInstance.get<stripeTypes.GetStripePaymentMethod[]>(END_POINTS.STRIPE_PAYMENTMETHOD_GET),
//     getPaidInvoiceList: () => axiosInstance.get<stripeTypes.GetPaidInvoice[]>(`${END_POINTS.STRIPE_PAIDINVOICE_GET}`),
//     // createCustomer: () => axiosInstance.post(END_POINTS.STRIPE_CUSTOMER),
//     createPaymentMethod: (body: stripeTypes.CreatePaymentMethodPayload) => axiosInstance.post<string>(END_POINTS.STRIPE_PAYMENTMETHOD_CREATE, helper.objectToFormData(body)),
//     createSubscription: (body) => axiosInstance.post<string>(END_POINTS.STRIPE_SUBSCRIPTION_CREATE, body),

//     createSubscriptionWithDefault: (body: stripeTypes.CreateSubscriptionWithDefaultPM) => axiosInstance.post<string>(END_POINTS.STRIPE_SUBSCRIPTION_CREATE_DEFAULT, helper.objectToFormData(body)),
//     updateSubscription: (body: stripeTypes.UpdateStripeSubscription) => axiosInstance.post(END_POINTS.STRIPE_SUBSCRIPTION_UPDATE, helper.objectToFormData(body)),
//     cancelSubscription: (body) => axiosInstance.post(END_POINTS.STRIPE_SUBSCRIPTION_CANCEL, body),

// }

export const announcementsRequests = {
    getAnnouncements: () => axiosInstance.get(END_POINTS.ANNOUCEMENTS),
}

// Settings
export const settingsRequests = {
    getCompanySettings: (companyId: string) => axiosInstance.get(`${END_POINTS.SETTINGS}/${companyId}`),
    updateCompanySettings: (companyId: string, body: ICompanySettings) => axiosInstance.put(`${END_POINTS.SETTINGS}/${companyId}`, body),

    getServerDateTime: () => axiosInstance.get(`${END_POINTS.SETTINGS}/date-time`),
    importTestDBItems: (data) => axiosInstance.post(`${END_POINTS.SETTINGS}/test/items `, data),
}

//Sample
export const sampleRequests = {
    getUsers: () => axiosInstance.get(END_POINTS.GET_SAMPLE)
}


// Help Center
export const helpCenterRequests = {
    shareFeedback: (body) => axiosInstance.post(`${END_POINTS.SHARE_FEEDBACK}`, body),
}
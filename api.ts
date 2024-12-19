import Announcements from "../pages/announcements";

export let API_URL = import.meta.env.VITE_APP_API_URL;
export let SOCKET_URL = import.meta.env.VITE_APP_SOCKET_URL;

export const HEADERS = {
    TIMEOUT_LIMIT: 1000,
    BEARER: "Bearer ",
}

export const END_POINTS = {
    // sample
    GET_SAMPLE: "users",

    // authentication
    SIGN_IN: "signin",
    SIGN_OUT: "signout",
    RESEND_ACTIVATION: "resend_activation",
    SIGN_UP: "signup",
    REQUEST_PASSWORD_RESET: "request_password_reset",
    RESET_PASSWORD: "reset_password",
    CHANGE_PASSWORD: "change_password",
    ACTIVATE_USER: "users/activate",
    RESEND_EMAIL_RESET_PASSWORD: "resend_email_reset_password",
    CHECK_PASSWORD: "check_password",
    CHECK_TOKEN_VALIDITY: "token_validity",
    GOOGLE: "google",
    MICROSOFT: "ms_office",
    MIGRATE: "migrate",
    SSO: "sso", 
    TOKEN_LOGIN: "token_login",
    
    // people
    USERS: "users",
    GET_ALL_USERS: "users/all",
    UPDATE_EMAIL: "users/email",

    UPDATE_PROFILE: "users/profile",
    UPDATE_USER_PHOTO: "/users/profile/photo",
    UPDATE_USER_INFO: "/users/profile/updateinfo",
    UPDATE_ACCOUNT: "users/profile/account",
    UPDATE_ACCOUNTS: "users/accounts",
    CREATE_USER_NEW: "users/new",

    GET_USER_COMPANIES: "companies/user",
    SKIP_SETUP_WIZARD: "users/skip-wizard",
    FINISH_TOUR: "users/finishtour",
    CONFIRM_VERIFICATION: "users/confirm-verification",
    REMOVE_MULTIPLE_USERS: "users/remove/multiple",
    MANAGE_ACCESS: "users/manage-access",
    REQUEST_ACCOUNT_CREATION: "/request-account",
    REQUEST_MATCH_ACCOUNT: "/request-match-account",
    REQUEST_INVITE_ACCOUNT: "/request-invitation",

    // company
    GET_ALL_COMPANIES: "companies",
    GET_COMPANIES_BY_USER: "companies/user",
    COMPANIES: "companies",
    ACTIVE_COMPANY: "/companies/active",
    ADD_BOOKMARK_COMPANY: "companies/bookmark/add",
    DELETE_BOOKMARK_COMPANY: "companies/bookmark/delete",
    UPDATE_COMPANY_LOGO: "companies/logo",

    // department
    DEPARTMENTS: "departments",
    ADD_BOOKMARK_DEPARTMENTS: "departments/bookmark/add",
    DELETE_BOOKMARK_DEPARTMENTS: "departments/bookmark/delete",
    GET_ALL_DEPARTMENTS: "departments/all",
    UPDATE_DEPARTMENT_STATUS: "departments/status",

    // group
    GROUP: "groups/profile/",
    GROUPS: "groups",
    ADD_MEMBER: "groups/members/add/",
    REQUEST_TO_JOIN_GROUP: "groups/members/request/",
    DELETE_MEMBER: "groups/members/delete/",
    DELETE_MEMBER_GROUP: "groups/", //groups/:groupID/delete
    GET_ALL_GROUPS: "groups/all",
    GET_ALL_GROUPS_BY_COMPANYID: "groups/company/",
    CLONE_GROUP: "groups/clone",
    MERGE_GROUP: "groups/merge",
    BRANCH_GROUP: "groups/branch",
    DELETE_GROUP: "groups/delete",
    GET_ALL_GROUPS_COUNT: "/groups/company/count",
    GET_GROUP_MEMBERS_COUNT: "/groups/profile/count",

    // group members
    ADD_GROUP_MEMBER_USERS: "groups/add_members/individual",
    ADD_GROUP_MEMBER_GROUPS: "groups/add_members/group",
    UPDATE_MEMBER: "groups/members/types",
    CRON_JOB_GROUP_MEMBERS: "groups/members/cron_job",
    GROUP_INTEGRATIONS: "groups/integrations",
    MOVE_MEMBERS_TO_ANOTHER_GROUPS: "groups/members/move",

    //group integrations
    ADD_GROUP_INTEGRATION: "groups/integrations/add",
    DELETE_GROUP_INTEGRATION: "groups/integrations/delete",
    DELETE_GROUP_INTEGRATION_REQUEST: "groups/integrations/delete-request",

    //group bookmark
    ADD_BOOKMARK_GROUP: "groups/bookmark/add",
    DELETE_BOOKMARK_GROUP: "groups/bookmark/delete",

    SUGGESTED_GROUPS: "groups/suggested",
    ADD_USERS_TO_GROUPS: "groups/members/bulk/add",

    // group - connect integration to groups
    CONNECT_INTEGRATION_TO_GROUPS: "groups/integrations/connect",

    //roles
    GET_ROLE: "role/",
    GET_ALL_ROLES: "roles/",
    ADD_NEW_ROLE: "roles/",
    GET_NUM_USERS: "role/user/",
    DELETE_ROLE: "roles/delete/",
    ASSIGN_PERMISSIONS: "roles/update/",
    ASSIGN_ROLE: "roles/assign",
    UNASSIGN_ROLE: "roles/unassign",
    REQUEST_ROLES: "roles/request",

    // permissions
    GET_ALL_PERMISSIONS: "permissions",

    // log
    GET_ALL_LOGS: "logs/all",
    LOGS: "logs",

    // requests
    REQUESTS: "requests",

    // integrations
    INTEGRATIONS: "integrations",
    CONNECT_INTEGRATION: "integrations/connect",
    DISCONNECT_INTEGRATION: "integrations/disconnect",
    INTEGRATION_UIDS: "integrations/uids",
    BATCH_INTEGRATION_UIDS: "integrations/uids/batch",
    SUB_INTEGRATIONS: "sub-integrations",
    MIGRATE_INTEGRATIONS: "integrations/migrate",
    SEND_EMAIL_INTEGRATION_CREDENTIALS: "integrations/send-email-credentials",
    ALLOW_EXTERNAL_USERS: "integrations/allow-external-users",

    // notifications
    NOTIFICATIONS: "notifications",
    MY_NOTIFICATIONS: "notifications/me",
    SEEN_NOTIFICATION: "notifications/seen",

    // subscriptions
    SUBSCRIPTIONS: "subscriptions",
    SUBSCRIPTIONS_USERS: "subscriptions/users",
    SUBSCRIPTIONS_COMPANY: "company",
    CREATE_SUBSCRIPTIONS: "subscriptions/create",
    UPDATE_SUBSCRIPTIONS: "subscriptions/update",

    // action items
    ACTION_ITEMS: "action_items",

    //mailing
    INTEG_NEW_USER_MAIL: "users/integration/account/new",

    // google cloud
    GCLOUD_AUTH: "google-cloud/auth",
    GCLOUD_TOKEN: "google-cloud/token",
    GA_USERS: "google-cloud/google-admin/users",
    GA_GROUPS: "google-cloud/google-admin/groups",
    GCP_PROJECTS: "google-cloud/gcp/projects",
    GCP_TEST_IAM_PERMISSIONS: "google-cloud/gcp/test-permissions",
    GCP_ORGANIZATIONS: "google-cloud/gcp/organizations",
    GCLOUD_IAM: "google-cloud/iam",
    GCLOUD_PEOPLE_ME: "google-cloud/people/me",
    GDRIVE_DRIVES: "google-cloud/google-drive/drives",
    GDRIVE_FILES: "google-cloud/google-drive/files",
    GA_IMPORT_GROUPS: "google-cloud/google-admin/import/groups",
    GA_GROUPS_IMPORT: "google-cloud/google-admin/groups/import",
    GA_CUSTOMER: "google-cloud/google-admin/customer",

    //JIRA
    JIRA_ADD_KEY: "users/token/jira",
    JIRA_REMOVE_KEY: "users/token/jira/remove",

    //STRIPE
    STIPE_SUBSCRIPTION_STATUS: "stripe/subscription/status",
    STRIPE_CUSTOMER: "stripe/customer",
    STRIPE_SUBSCRIPTION_CREATE: "stripe/subscribe/create",
    STRIPE_SUBSCRIPTION_CREATE_DEFAULT: "stripe/subscribe/create/default",
    STRIPE_SUBSCRIPTION_UPDATE: "stripe/subscribe/update",
    STRIPE_SUBSCRIPTION_CANCEL: "stripe/subscribe/cancel",
    STRIPE_PAYMENTMETHOD_CREATE: "stripe/paymentmethod",
    STRIPE_PAYMENTMETHOD_GET: "stripe/paymentmethod/list",
    STRIPE_PAIDINVOICE_GET: "stripe/invoice/paid/list",

    //ANNOUCEMENTS
    ANNOUCEMENTS: "Announcements",
    // Settings
    SETTINGS: "settings",

    // Help Center
    SHARE_FEEDBACK: "help-center/share-feedback",

    //New Logs
    NEW_LOGS: "new/logs",
}
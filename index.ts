import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { ENTITY_TYPE, ITEM_STATUS, LOG_ACTION } from "../../../../utils/constants";
import helper from "../../../../utils/helper";


const excludeUsersWithStatusOnRestore = [ITEM_STATUS.PENDING, ITEM_STATUS.ACTIVE, ITEM_STATUS.DEFAULT];

const isCurrentUser = (userId: string | undefined, currentUser: any) => (currentUser?.UserID == userId);

const transformName = (user: { ID?: string, Name?: string }, currentUser: any):string | undefined => {
    return isCurrentUser(user?.ID, currentUser) ? "you" : user?.Name;
}

const listUsersToString = (users: any, currentUser: any): string => {
    return users?.map(user =>  isCurrentUser(user?.ID, currentUser) ? "you" : user?.Name).toString();
}

const getVerb = (items: any, verb: string, currentUser: any) => {
    const count = items?.length;
    if (!count) return verb;
    if (verb == "has") return count == 1 ? isCurrentUser(items?.[0]?.ID, currentUser) ? "have" : "has" : "have";
    if (verb == "doesn't") return count == 1 ? isCurrentUser(items?.[0]?.ID, currentUser) ? "don't" : "doesn't" : "don't";
    return verb;
}

const getAuthor = (activity: any): string => {
    if (activity?.LogInfo?.Author?.Status == ITEM_STATUS.PERMANENTLY_DELETED) return `, by ${activity?.LogInfo?.Author?.Name} (Removed)`;
    return `, by ${activity?.LogInfo?.Author?.Name}`;
}

export const formatActivitiesDescriptionPlainText = (activity: any, currentUser: any):string => {
    dayjs.extend(relativeTime);
    if (activity.LogType === ENTITY_TYPE.AUTH) {
        switch (activity.LogAction) {
            case LOG_ACTION.SIGN_UP:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "" : "has"} registered on the application. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.ESTABLISH_COMPANY:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "" : "has"} created the ${activity?.LogInfo?.Company?.ID} ${activity.LogInfo?.Company?.Name} (Company). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.SIGN_OUT:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "" : "has"} logged out from the application. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.SIGN_IN:
               return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "" : "has"} logged in to the application. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REQUEST_PASSWORD_RESET:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "" : "has"} requested to change ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "your" : "has"} password. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.PASSWORD_CHANGED:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity.LogInfo?.User?.ID, currentUser) ? "" : "has"} changed ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "your" : "their"} password. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.ACTIVATE_ACCOUNT:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity.LogInfo?.User?.ID, currentUser) ? "" : "has"} activated ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "your" : "their"} account. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    if (activity.LogType === ENTITY_TYPE.COMPANY) {
        switch (activity.LogAction) {
            case LOG_ACTION.ADD_COMPANY:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity.LogInfo?.User?.ID, currentUser) ? "" : "has"} created the ${activity?.LogInfo?.Company?.Name} company. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.UPDATE_COMPANY:
               return `${activity?.LogInfo?.Company?.Name} Company information has been updated. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    if (activity.LogType === ENTITY_TYPE.COMPANYMEMBER) {
        const oldUsers = activity?.LogInfo?.Users?.map(user => ({ Name: user?.Temp?.Name })) || [];
        switch (activity.LogAction) {
            case LOG_ACTION.ADD_COMPANY_MEMBERS:
                return `${listUsersToString(activity?.LogInfo?.Users, currentUser)} ${getVerb(activity?.LogInfo?.Users, "has", currentUser)} been added to ${activity?.LogInfo?.Company?.Name} Company. ${activity?.ActionItemID ? "Would you like to add this user to a group?" : ""}. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_COMPANY_MEMBERS:
                return `${listUsersToString(oldUsers, currentUser)} ${getVerb(oldUsers, "has", currentUser)} been removed from ${activity?.LogInfo?.Company?.Name} Company. ${(activity?.ActionItemID && !excludeUsersWithStatusOnRestore?.includes(activity?.Log?.LogInfo?.Users?.[0]?.Temp?.Status)) ? "Would you like to restore this user?" : ""}. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.RESTORE_COMPANY_MEMBERS:
                return `${listUsersToString(activity?.LogInfo?.Users, currentUser)} ${getVerb(activity?.LogInfo?.Users, "has", currentUser)} been restored from ${activity?.LogInfo?.Company?.Name} (Company). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.PERMANENTLY_REMOVE_COMPANY_MEMBERS:
               return `${listUsersToString(oldUsers, currentUser)} ${getVerb(oldUsers, "has", currentUser)} been permanently removed from ${activity?.LogInfo?.Company?.Name} company. ${(activity?.ActionItemID && !excludeUsersWithStatusOnRestore?.includes(activity?.Log?.LogInfo?.Users?.[0]?.Temp?.Status)) ? "Would you like to restore this user?" : ""}. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.INVITE_REMOVE_COMPANY_MEMBER:
               return `${activity?.LogInfo?.User?.Name} hasn't responded to your invite sent. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_COMPANY_MEMBERS_INTEGRATION_ACCESS:
                return `${activity?.LogInfo?.Integration?.Name} access for ${listUsersToString(oldUsers, currentUser)} has been removed. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    if (activity.LogType === ENTITY_TYPE.GROUP) {
        switch (activity.LogAction) {
            case LOG_ACTION.ADD_GROUP:
               return `${activity?.LogInfo?.Group?.Name} Group has been added to ${activity?.LogInfo?.Department?.Name} (Department). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.UPDATE_GROUP:
               return `${activity?.LogInfo?.Group?.Name} Group information has been updated. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.ADD_GROUP_INTEGRATION:
                return `${activity?.LogInfo?.Integration?.Name} has been connected to ${activity?.LogInfo.Group?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.DELETE_GROUP:
                return `${activity?.LogInfo?.Group?.Name} Group has been removed from ${activity?.LogInfo?.Department?.Name} (Department). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_GROUP:
                return `${activity?.LogInfo?.Group?.Name} Group has been removed from ${activity?.LogInfo?.Department?.Name} (Department). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.CLONE_GROUP:
               return `${activity?.LogInfo?.Group?.Name} Group has been cloned from ${activity?.LogInfo?.Origin?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.MERGE_GROUP:
              return `${activity?.LogInfo?.RemovedGroup?.Name} Group has been merged into ${activity?.LogInfo?.RetainedGroup?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_GROUP_INTEGRATION:
                return `${activity?.LogInfo?.Integration?.Name} has been disconnected from ${activity?.LogInfo?.Group?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    if (activity?.LogType === ENTITY_TYPE.GROUPMEMBER) {
        switch (activity?.LogAction) {
            case LOG_ACTION.ADD_GROUP_MEMBERS:
                if (activity?.LogInfo?.Users?.length && activity?.LogInfo.Users?.[0]?.Name) {
                    return `${listUsersToString(activity?.LogInfo?.Users, currentUser)} ${getVerb(activity?.LogInfo?.Users, "has", currentUser)} been added to ${activity?.LogInfo?.Group?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
                } else {
                   return `${listUsersToString(activity?.LogInfo?.Groups, currentUser)} ${getVerb(activity?.LogInfo?.Users, "has", currentUser)} been added to ${activity.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity?.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : activity?.LogInfo?.Group?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
                }
            case LOG_ACTION.BRANCH_GROUP:
                return `${activity?.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity?.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : activity.LogInfo?.Group?.Name} Group has been branched from ${activity?.LogInfo?.Origin?.Status === ITEM_STATUS.INACTIVE ? activity?.LogInfo?.Origin?.Name : activity.LogInfo?.Origin?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_INDIVIDUAL_MEMBERS:
                return `${listUsersToString(activity?.LogInfo?.Users, currentUser)} ${getVerb(activity?.LogInfo?.Users, "has", currentUser)} been removed from ${activity.LogInfo?.Group?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_GROUP_MEMBERS:
                return `${listUsersToString(activity?.LogInfo?.Users, currentUser)} getVerb(activity?.LogInfo?.Members, "has", currentUser)} been removed from ${activity?.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : activity?.LogInfo?.Group?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.DELETE_GROUP_MEMBERS:
                return `${listUsersToString(activity?.LogInfo?.Users, currentUser)} getVerb(activity?.LogInfo?.Members, "has", currentUser)} been deleted from ${activity?.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity?.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : activity?.LogInfo?.Group?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.MOVE_GROUP_MEMBERS:
                return `${listUsersToString(activity?.LogInfo?.Users, currentUser)} ${getVerb(activity?.LogInfo?.Users, "has", currentUser)} been moved from ${activity.LogInfo?.SourceGroup?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.SourceGroup?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.SourceGroup?.Name : activity?.LogInfo?.SourceGroup?.Name} (Group) to ${activity.LogInfo?.DestinationGroup?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.DestinationGroup?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.DestinationGroup?.Name : activity?.LogInfo?.DestinationGroup?.Name} (Group). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
            break;
        }
    }

    if (activity.LogType === ENTITY_TYPE.USER) {
        switch (activity.LogAction) {
            case LOG_ACTION.CLONE_USER_GROUPS:
                return `${activity?.LogInfo?.User?.Name} ${isCurrentUser(activity.LogInfo?.User?.ID, currentUser) ? "have" : "has"} been added to groups. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.UPDATE_USER:
               return `${currentUser?.UserID == activity?.LogInfo?.User?.ID ? ((currentUser?.UserID == activity?.LogInfo?.Author?.ID) ? "You" : "Your") : (activity?.LogInfo?.User?.ID == activity.LogInfo?.Author?.ID ? activity?.LogInfo?.User?.Name : `${activity?.LogInfo?.User?.Name}'s`)}. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    if (activity.LogType === ENTITY_TYPE.INTEGRATION) {
        switch (activity.LogAction) {
            case LOG_ACTION.CONNECT_INTEGRATION:
                return `You've added ${activity?.LogInfo?.Integration?.Name} to your list of integration. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.DISCONNECT_INTEGRATION:
                return `${activity?.LogInfo?.Integration?.Name} has been disconnected from ${activity?.LogInfo?.Company?.Name} (Company). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REQUEST_INTEGRATION:
                return `${transformName(activity?.LogInfo?.User, currentUser)} ${isCurrentUser(activity.LogInfo?.User?.ID, currentUser) ? "" : "has"} requested to support a new integration. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    if (activity.LogType === ENTITY_TYPE.ROLE) {
        switch (activity.LogAction) {
            case LOG_ACTION.ADD_ROLE:
                return `${activity?.LogInfo?.User?.Name} ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "" : "has"} created ${activity?.LogInfo?.Role?.Name} role. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.DELETE_ROLE:
                return `${activity?.LogInfo?.User?.Name} ${isCurrentUser(activity.LogInfo?.User?.ID, currentUser) ? "" : "has"} deleted ${activity.LogInfo?.Role?.Name} role. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_ROLE:
                return `${activity?.LogInfo?.User?.Name} ${isCurrentUser(activity?.LogInfo?.User?.ID, currentUser) ? "" : "has"} removed ${activity?.LogInfo?.Role?.Name} role. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.UPDATE_ROLE:
                return `${activity?.LogInfo?.User?.Name} ${isCurrentUser(activity.LogInfo?.User?.ID, currentUser) ? "" : "has"} updated ${activity.LogInfo?.Role?.Name} role. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    if (activity.LogType === ENTITY_TYPE.DEPARTMENT) {
        switch (activity.LogAction) {
            case LOG_ACTION.ADD_DEPARTMENT:
                return `${activity?.LogInfo?.Department?.Name} Department has been added to ${activity.LogInfo?.Company?.Name} (Company). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.UPDATE_DEPARTMENT:
                return `${activity?.LogInfo?.Department?.Name} Department information has been updated. ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.DELETE_DEPARTMENT:
                return `${activity?.LogInfo?.Department?.Name} Department has been removed from ${activity?.LogInfo?.Company?.Name} (Company). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            case LOG_ACTION.REMOVE_DEPARTMENT:
                return `${activity?.LogInfo?.Department?.Name} Department has been removed from ${activity?.LogInfo?.Company?.Name} (Company). ${helper.formatDateWithFromNow(activity?.CreatedAt)} ${getAuthor(activity)}`;
            default:
                break;
        }
    }

    return "";
}



//1.1 React, packages, librariesxxxxx
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import React from "react";
import { connect } from "react-redux";
import { Link } from "react-router-dom";
//1.2 Redux (made by team)
import { AppState } from "../../../../../redux/reducers";
//1.3 Pages/Components (made by team)
import BulkEntry from "./bulk-entry";
//1.4.1 Interface
//1.4.2 Types
//1.4.3 Helper
//1.4.4 Hooks
//1.4.5 Constants
//1.4.6 Utils
import { ACTION_ITEM_TYPE, ENTITY_TYPE, ITEM_STATUS, LOG_ACTION } from "../../../../../utils/constants";
import { excludeUsersWithStatusOnRestore } from "../action-buttons";
import StatusBadgeTooltip from "../../../../groups-info/components/integrations/components/tooltips/status-badge";
import { Badge } from "react-bootstrap";
import { ACTION_PRIORITY } from "../../../../../utils/constants";
import PriorityIndicator from "../priority-indicator";
import helper from "../../../../../utils/helper";
//1.4.7 Services
//1.4.8 Syles
interface DescriptionProps {
    activity,
    actionItem?: string,
    actionItemType?: string,
    actionItemData?,
    currentUser?,
    groupId?: string,
}


const Description: React.FC<DescriptionProps> = (props) => {
    const { activity, actionItem, actionItemType } = props;
    const { actionItemData } = props;
    const { currentUser, groupId } = props;

    const transformCategoryName = (category: string) => {
        if (!category) return;
        category = category.toLowerCase();
        const lWord = category.split(" ").pop();
        const lIndex = category.lastIndexOf(" ");
        if (lWord == "app" || lWord == "apps") {
            category = category.substring(0, lIndex);
        }
        return category + " apps";
    }

    const getVerb = (items, verb: string) => {
        const count = items?.length;
        if (!count) return verb;
        if (verb == "has") return count == 1 ? isCurrentUser(items?.[0]?.ID) ? "have" : "has" : "have";
        if (verb == "doesn't") return count == 1 ? isCurrentUser(items?.[0]?.ID) ? "don't" : "doesn't" : "don't";
        return verb;
    }

    const isCurrentUser = (userId: string | undefined) => (currentUser?.UserID == userId);

    const transformName = (user: { ID?: string, Name?: string }) => {
        if (isCurrentUser(user?.ID)) return <span>You</span>;
        return <span>{user?.Name}</span>;
    }

    const customIntegrationMessage = () => {
        const integration = activity.LogInfo?.Integration;
        
        const integrationLink = (
            <Link className="font-weight-bolder" to={`/integrations/${activity.LogInfo?.Integration?.ID}`}>
                {activity.LogInfo?.Integration?.Name}
            </Link>
        );

        let integText: any;
        switch (integration?.Name) {
            case "Slack":
                integText = <>Integration for {integrationLink}</>
                break;
            default:
                integText = <>{integrationLink}</>
                break;
        }
        //default integ text = <><Link className="font-weight-bolder" to={`/integrations/${activity.LogInfo?.Integration?.ID}`}>{activity.LogInfo?.Integration?.Name}</Link> (Integration) has been connected to <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group.ID}`}>{activity.LogInfo.Group.Name} </Link> (Group).</>;
        return integText;
    };

    const suggestIntegrationDescription = (activity: any, actionItemType: string):string => {
        
        if (actionItemType === ACTION_ITEM_TYPE.SUGGEST_INTEGRATION_CATEGORY) {
            return `Would you like to also add ${(activity.LogInfo?.Integration?.Categories?.length === 1 && transformCategoryName(activity.LogInfo?.Integration?.Categories?.[0])) || "apps"}?`;
        } else if (actionItemType === ACTION_ITEM_TYPE.SUGGEST_HOT_INTEGRATION) {
            return `Would you like to also add ${activity.LogInfo?.SuggestedIntegration?.Name}?`;
        } else if (actionItemType === ACTION_ITEM_TYPE.IMPORT_INTEGRATION_CONTACTS) {
            return "Would you like to import your contacts?";
        } else {
            return "";
        }
    }


    const formatActivityDescription = (activity) => {
        let text: JSX.Element = <></>;
        if (activity.LogType === ENTITY_TYPE.AUTH) {
            switch (activity.LogAction) {
                case LOG_ACTION.SIGN_UP:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} registered on the application.</>;
                    break;
                case LOG_ACTION.ESTABLISH_COMPANY:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} created the <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company).</>;
                    break;
                case LOG_ACTION.SIGN_OUT:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} logged out from the application.</>;
                    break;
                case LOG_ACTION.SIGN_IN:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} logged in to the application.</>;
                    break;
                case LOG_ACTION.REQUEST_PASSWORD_RESET:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} requested to change {isCurrentUser(activity.LogInfo?.User?.ID) ? "your" : "has"} password.</>;
                    break;
                case LOG_ACTION.PASSWORD_CHANGED:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} changed {isCurrentUser(activity.LogInfo?.User?.ID) ? "your" : "their"} password.</>;
                    break;
                case LOG_ACTION.ACTIVATE_ACCOUNT:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} activated {isCurrentUser(activity.LogInfo?.User?.ID) ? "your" : "their"} account.</>;
                    break;
                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.COMPANY) {
            switch (activity.LogAction) {
                case LOG_ACTION.ADD_COMPANY:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} created the <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company).</>;
                    break;
                case LOG_ACTION.UPDATE_COMPANY:
                    text = <><Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company) information has been updated.</>;
                    break;
                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.COMPANYMEMBER) {
            const oldUsers = activity?.LogInfo?.Users?.map(user => ({ Name: user?.Temp?.Name })) || [];
            switch (activity.LogAction) {
                case LOG_ACTION.ADD_COMPANY_MEMBERS:
                    text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{getVerb(activity?.LogInfo?.Users, "has")} been added to <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company). {actionItem ? "Would you like to add this user to a group?" : ""}</>;
                    break;
                case LOG_ACTION.REMOVE_COMPANY_MEMBERS:
                    text = <><BulkEntry module="users" entries={oldUsers} />{getVerb(oldUsers, "has")} been removed from <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company). {(actionItem && !excludeUsersWithStatusOnRestore?.includes(actionItemData?.Log?.LogInfo?.Users?.[0]?.Temp?.Status)) ? "Would you like to restore this user?" : ""}</>;
                    break;

                case LOG_ACTION.RESTORE_COMPANY_MEMBERS:
                    text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{getVerb(activity?.LogInfo?.Users, "has")} been restored from <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company).</>;
                    break;

                case LOG_ACTION.PERMANENTLY_REMOVE_COMPANY_MEMBERS:
                    text = <><BulkEntry module="users" entries={oldUsers} />{getVerb(oldUsers, "has")} been permanently removed from <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company). {(actionItem && !excludeUsersWithStatusOnRestore?.includes(actionItemData?.Log?.LogInfo?.Users?.[0]?.Temp?.Status)) ? "Would you like to restore this user?" : ""}</>;
                    break;

                case LOG_ACTION.INVITE_REMOVE_COMPANY_MEMBER:
                    dayjs.extend(relativeTime);
                    // const createdAt = actionItemData?.CreatedAt ? dayjs.unix(actionItemData?.CreatedAt).format("MMMM D, YYYY h:mm a") : dayjs.unix(activity?.CreatedAt).format("MMMM D, YYYY h:mm a");
                    // text = <>{activity?.LogInfo?.User?.Name} hasn't responded to your invite sent at {helper.formatDateWithFromNow(createdAt, actionItemData?.Extras?.CreatedAt || activity?.LogInfo?.User?.CreatedAt)}.</>;
                    const createdAt = actionItemData?.CreatedAt ? dayjs.unix(actionItemData?.CreatedAt).format("YYYY-MM-DD") : dayjs.unix(activity?.CreatedAt).format("YYYY-MM-DD");
                    text = <>{activity?.LogInfo?.User?.Name} hasn't responded to your invite sent {dayjs.unix(actionItemData?.Extras?.CreatedAt || activity?.LogInfo?.User?.CreatedAt).from(createdAt)}.</>;
                    break;

                case LOG_ACTION.REMOVE_COMPANY_MEMBERS_INTEGRATION_ACCESS:
                    text = <>{customIntegrationMessage()} access for <BulkEntry module="users" entries={oldUsers} /> has been removed.</>
                    break;

                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.GROUP) {
            switch (activity.LogAction) {
                case LOG_ACTION.ADD_GROUP:
                    text = <><Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group) has been added to <Link className="font-weight-bolder" to={`/departments/${activity.LogInfo?.Department?.ID}`}>{activity.LogInfo?.Department?.Name}</Link> (Department).</>;
                    break;
                case LOG_ACTION.UPDATE_GROUP:
                    text = <><Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group) information has been updated.</>;
                    break;
                case LOG_ACTION.ADD_GROUP_INTEGRATION:
                    text = <>{customIntegrationMessage()} has been connected to <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group.ID}`}>{activity.LogInfo.Group.Name} </Link> (Group).</>;
                    break;
                case LOG_ACTION.DELETE_GROUP:
                    // 
                    text = <><span className="text-primary" >{activity.LogInfo?.Group?.Name} </span>(Group) has been removed from <Link className="font-weight-bolder" to={`/departments/${activity.LogInfo?.Department.ID}`}>{activity.LogInfo?.Department.Name} </Link>(Department).</>;
                    break;
                case LOG_ACTION.REMOVE_GROUP:
                    text = <><span className="text-primary" >{activity.LogInfo?.Group?.Name} </span>(Group) has been removed from <Link className="font-weight-bolder" to={`/departments/${activity.LogInfo?.Department?.ID}`}>{activity.LogInfo?.Department.Name} </Link>(Department).</>;
                    break;
                case LOG_ACTION.CLONE_GROUP:
                    text = <><Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/users/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group) has been cloned from <Link className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Origin.ID}`}>{activity.LogInfo?.Origin?.Name}</Link> (Group).</>;
                    // See the new member{activity.LogInfo?.Members?.length > 1 && "s"} of group<Link to={`/groups/${activity.LogInfo?.Group?.ID}`}> {activity.LogInfo?.Group?.Name}</Link>.
                    break;
                case LOG_ACTION.MERGE_GROUP:
                    // text = <>Group <Link className="font-weight-bolder" to={`/groups/${activity.LogInfo?.RemovedGroup.ID}`}>{activity.LogInfo?.RemovedGroup.Name}</Link> has been merged into  Group <Link className="font-weight-bolder" to={`/groups/${activity.LogInfo?.RetainedGroup?.ID}`}>{activity.LogInfo?.RetainedGroup?.Name}</Link>. <BulkEntry module="members" entries={activity?.LogInfo?.Members} />{activity.LogInfo?.Members?.length == 1 ? "is" : " are"} the new member{activity.LogInfo?.Members?.length > 1 && "s"} of <Link to={`/groups/${activity.LogInfo?.RetainedGroup?.ID}`}>{activity.LogInfo?.RetainedGroup?.Name}</Link>.</>;
                    text = <><Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.RemovedGroup.ID}`}>{activity.LogInfo?.RemovedGroup?.Name} </Link>(Group) has been merged into <Link className="font-weight-bolder" to={`/groups/${activity.LogInfo?.RetainedGroup?.ID}`}>{activity.LogInfo?.RetainedGroup?.Name}</Link> (Group).</>;
                    // See the new member{activity.LogInfo?.Members?.length > 1 && "s"} of group<Link to={`/groups/${activity.LogInfo?.RetainedGroup?.ID}`}> {activity.LogInfo?.RetainedGroup?.Name}</Link>.
                    break;
                // case LOG_ACTION.ADD_BOOKMARK_GROUP:
                //     text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} (Group)</Link> has been bookmarked.</>;
                //     break;  
                // case LOG_ACTION.DELETE_BOOKMARK_GROUP:
                //     text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.User?.ID}`}>{activity.LogInfo?.User?.Name} (Group)</Link> has removed bookmark from group <span className="text-primary" >{activity.LogInfo?.Group?.Name}</span>.</>;
                //     break;
                case LOG_ACTION.REMOVE_GROUP_INTEGRATION:
                    text = <>{customIntegrationMessage()} has been disconnected from <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group.ID}`}>{activity.LogInfo?.Group?.Name}</Link> (Group).</>;
                    break;
                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.GROUPMEMBER) {
            switch (activity.LogAction) {
                case LOG_ACTION.ADD_GROUP_MEMBERS:
                    if (activity.LogInfo?.Users?.length && activity.LogInfo.Users?.[0]?.Name) {
                        // text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{getVerb(activity?.LogInfo?.Users, "has")} been added to <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group). {actionItem ? `Would you like to add ${(Object.keys(activity?.LogInfo?.Users || [])?.length > 1) ? 'these users' : 'this user'} to other groups?` : ""}</>;
                        text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{getVerb(activity?.LogInfo?.Users, "has")} been added to {activity.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>} (Group). {activity?.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity?.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? <></> : actionItem ? `Would you like to add ${(Object.keys(activity?.LogInfo?.Users || [])?.length > 1) ? 'these users' : 'this user'} to other groups?` : ""}</>;
                    } else {
                        // text = <><BulkEntry module="groups" entries={activity?.LogInfo?.Groups} />{getVerb(activity?.LogInfo?.Users, "has")} been added to <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group). {actionItem ? `Would you like to add ${(Object.keys(activity?.LogInfo?.Groups || [])?.length > 1) ? 'these groups' : 'this group'} to other groups?` : ""}</>;
                        text = <><BulkEntry module="groups" entries={activity?.LogInfo?.Groups} />{getVerb(activity?.LogInfo?.Users, "has")} been added to {activity.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>}(Group). {activity?.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity?.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? <></> : actionItem ? `Would you like to add ${(Object.keys(activity?.LogInfo?.Groups || [])?.length > 1) ? 'these groups' : 'this group'} to other groups?` : ""}</>;
                    }
                    break;
                case LOG_ACTION.BRANCH_GROUP:
                    // text = <><Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/users/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group) has been branched from <Link className="font-weight-bolder" to={`/groups/${activity?.LogInfo?.Origin?.ID}`}>{activity.LogInfo?.Origin?.Name} </Link>(Group).</>;
                    text = <>{activity.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>}(Group) has been branched from {activity.LogInfo?.Origin?.Status === ITEM_STATUS.INACTIVE ? activity?.LogInfo?.Origin?.Name : <Link style={{ cursor: `${activity.LogInfo?.Origin?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Origin?.ID}`}>{activity.LogInfo?.Origin?.Name} </Link>}(Group).</>;
                    // See the new member{activity.LogInfo?.Members?.length > 1 && "s"} of <Link to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group).
                    break;
                case LOG_ACTION.REMOVE_INDIVIDUAL_MEMBERS:
                    // text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{(actionItem) ? `${getVerb(activity?.LogInfo?.Users, "doesn't")} belong to any group. Would you like to add this user to a group?` : `${getVerb(activity?.LogInfo?.Users, "has")} been removed from `} {(actionItem) ? "" : <><Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group).</>}</>;
                    text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{(actionItem) ? `${getVerb(activity?.LogInfo?.Users, "doesn't")} belong to any group. Would you like to add this user to a group?` : `${getVerb(activity?.LogInfo?.Users, "has")} been removed from `} {(actionItem) ? "" : <>{activity.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>}(Group).</>}</>;
                    break;
                case LOG_ACTION.REMOVE_GROUP_MEMBERS:
                    // text = <><BulkEntry module="groups" entries={activity?.LogInfo?.Members} />{getVerb(activity?.LogInfo?.Members, "has")} been removed from <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group).</>;
                    text = <><BulkEntry module="groups" entries={activity?.LogInfo?.Members} />{getVerb(activity?.LogInfo?.Members, "has")} been removed from {activity.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>}(Group).</>;
                    break;
                case LOG_ACTION.DELETE_GROUP_MEMBERS:
                    // text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{getVerb(activity?.LogInfo?.Members, "has")} been deleted from <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>(Group).</>;
                    text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />{getVerb(activity?.LogInfo?.Members, "has")} been deleted from {activity.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || activity.LogInfo?.Group?.Status === ITEM_STATUS.DELETED ? activity?.LogInfo?.Group?.Name : <Link style={{ cursor: `${activity.LogInfo?.Group?.ID === groupId ? "default" : "pointer"}` }} className="font-weight-bolder" to={`/groups/${activity.LogInfo?.Group?.ID}`}>{activity.LogInfo?.Group?.Name} </Link>}(Group).</>;
                    break;
                    case LOG_ACTION.MOVE_GROUP_MEMBERS:
                        text = <><BulkEntry module="users" entries={activity?.LogInfo?.Users} />
                            {getVerb(activity?.LogInfo?.Users, "has")} been moved from {' '}
                            {activity.LogInfo?.SourceGroup?.Status === ITEM_STATUS.INACTIVE || 
                             activity.LogInfo?.SourceGroup?.Status === ITEM_STATUS.DELETED ? 
                                activity?.LogInfo?.SourceGroup?.Name : 
                                <Link style={{ cursor: `${activity.LogInfo?.SourceGroup?.ID === groupId ? "default" : "pointer"}` }} 
                                      className="font-weight-bolder" 
                                      to={`/groups/${activity.LogInfo?.SourceGroup?.ID}`}>
                                    {activity.LogInfo?.SourceGroup?.Name}
                                </Link>
                            } (Group) to {' '}
                            {activity.LogInfo?.DestinationGroup?.Status === ITEM_STATUS.INACTIVE || 
                             activity.LogInfo?.DestinationGroup?.Status === ITEM_STATUS.DELETED ? 
                                activity?.LogInfo?.DestinationGroup?.Name :
                                <Link style={{ cursor: `${activity.LogInfo?.DestinationGroup?.ID === groupId ? "default" : "pointer"}` }}
                                      className="font-weight-bolder" 
                                      to={`/groups/${activity.LogInfo?.DestinationGroup?.ID}`}>
                                    {activity.LogInfo?.DestinationGroup?.Name}
                                </Link>
                            } (Group).</>;
                        break;
                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.USER) {
            switch (activity.LogAction) {
                case LOG_ACTION.CLONE_USER_GROUPS:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.User?.ID}`}>{activity.LogInfo?.User?.Name}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "have" : "has"} been added to <BulkEntry module="groups" entries={activity?.LogInfo?.Groups || []} />.</>;
                    break;
                case LOG_ACTION.UPDATE_USER:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.User?.ID}`}>
                        {currentUser?.UserID == activity.LogInfo?.User?.ID ? ((currentUser?.UserID == activity.LogInfo?.Author?.ID) ? "You" : "Your") : (activity.LogInfo?.User?.ID == activity.LogInfo?.Author?.ID ? activity.LogInfo?.User?.Name : `${activity.LogInfo?.User?.Name}'s`)}
                    </Link> {activity.LogInfo?.User?.ID == activity.LogInfo?.Author?.ID
                        ? (currentUser?.UserID == activity.LogInfo?.Author?.ID) ? "updated your information." : "updated their information."
                        : "information was updated."} </>;
                    break;
                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.INTEGRATION) {
            switch (activity.LogAction) {
                case LOG_ACTION.CONNECT_INTEGRATION:
                    // 
                    text = <>You've added {activity.LogInfo?.Integration?.Name} to your list of integration. {suggestIntegrationDescription(activity,actionItemType!)}</>;
                    break;
                case LOG_ACTION.DISCONNECT_INTEGRATION:
                    text = <>{customIntegrationMessage()} has been disconnected from <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company).</>;
                    break;
                case LOG_ACTION.REQUEST_INTEGRATION:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo.User?.ID}`}>{transformName(activity.LogInfo?.User)}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} requested to support a new integration.</>;
                    break;
                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.ROLE) {
            // 
            switch (activity.LogAction) {
                case LOG_ACTION.ADD_ROLE:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.User?.ID}`}>{activity.LogInfo?.User?.Name}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} created <span className="text-primary" >{activity.LogInfo?.Role?.Name}</span> role.</>;
                    break;
                case LOG_ACTION.DELETE_ROLE:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.User?.ID}`}>{activity.LogInfo?.User?.Name}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} deleted <span className="text-primary" >{activity.LogInfo?.Role?.Name}</span> role.</>;
                    break;
                case LOG_ACTION.REMOVE_ROLE:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.User?.ID}`}>{activity.LogInfo?.User?.Name}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} removed <span className="text-primary" >{activity.LogInfo?.Role?.Name}</span> role.</>;
                    break;
                case LOG_ACTION.UPDATE_ROLE:
                    text = <><Link className="font-weight-bolder" to={`/users/${activity.LogInfo?.User?.ID}`}>{activity.LogInfo?.User?.Name}</Link> {isCurrentUser(activity.LogInfo?.User?.ID) ? "" : "has"} updated <span className="text-primary" >{activity.LogInfo?.Role?.Name}</span> role.</>;
                    break;
                default:
                    break;
            }
        }
        if (activity.LogType === ENTITY_TYPE.DEPARTMENT) {
            switch (activity.LogAction) {
                case LOG_ACTION.ADD_DEPARTMENT:
                    text = <><Link className="font-weight-bolder" to={`/departments/${activity.LogInfo?.Department?.ID}`}>{activity.LogInfo?.Department?.Name} </Link>(Department) has been added to <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company).</>;
                    break;
                case LOG_ACTION.UPDATE_DEPARTMENT:
                    text = <><Link className="font-weight-bolder" to={`/departments/${activity.LogInfo?.Department?.ID}`}>{activity.LogInfo?.Department?.Name} </Link>(Department) information has been updated.</>;
                    break;
                case LOG_ACTION.DELETE_DEPARTMENT:
                    text = <><span className="text-primary">{activity?.LogInfo?.Department?.Name} </span>(Department) has been removed from <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company).</>;
                    break;
                case LOG_ACTION.REMOVE_DEPARTMENT:
                    text = <><span className="text-primary">{activity?.LogInfo?.Department?.Name} </span>(Department) has been removed from <Link className="font-weight-bolder" to={`/companies/${activity.LogInfo?.Company?.ID}`}>{activity.LogInfo?.Company?.Name} </Link>(Company).</>;
                    break;
                default:
                    break;
            }
        }
        return text;
    }
    return (
        <p className="mb-0">
            {formatActivityDescription(activity)}
        </p>
    )
}

const mapStateToProps = (state: AppState) => ({
    currentUser: state?.Auth?.user,
});

const connector = connect(mapStateToProps, null);
export default connector(Description);
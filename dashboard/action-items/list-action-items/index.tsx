import React from "react";
import { ListGroup, OverlayTrigger, Tooltip } from "react-bootstrap";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { Link } from "react-router-dom";

import ActionButtons from "../../../activities/components/activity-item/action-buttons";
import Description from "../../../activities/components/activity-item/description";
import ActivityAvatar from "../../../activities/components/activity-item/avatar";
import { ITEM_STATUS, LOG_ACTION } from "../../../../utils/constants";
import { AppState } from "../../../../redux/reducers";
import { connect } from "react-redux";
import PriorityIndicator from "../../../activities/components/activity-item/priority-indicator";
import helper from "../../../../utils/helper";
import SourceIndicator from "../../../../components/source-indicator";
interface ActivityItemProps {
    activity,
    actionItem?: string,
    actionItemType?: string,
    onAddToGroup?: () => void,
    onAddIntegration: (filter: string[]) => void,
    onRemoveActionItem?: (actionItemId: string) => void,
    onImportUsers?,
    actionItemData?,
    onRestoreUser?: () => void,

    onInviteUser?: () => void,
    onRemoveUser?: () => void,

    currentUser?,
}

const ActivityItem: React.FC<ActivityItemProps> = (props) => {
    const {
        activity,
        actionItem,
        onAddToGroup,
        onRemoveActionItem,
        onAddIntegration,
        actionItemType,
        onImportUsers,
        actionItemData,
        currentUser,
    } = props;

    const { onRestoreUser } = props;
    const { onInviteUser, onRemoveUser } = props;

    const handleRemoveActionItem = () => {
        onRemoveActionItem && onRemoveActionItem(actionItem || "");
    }

    const getLogCreatedAt = () => {
        dayjs.extend(relativeTime);
        const now = dayjs();
        return (
            <span className="text-muted font-size-sm">{helper.formatDateWithFromNow(actionItemData?.CreatedAt || activity?.CreatedAt)}</span>
        )
    }

    const getAuthor = () => {
        if (actionItem || !activity?.LogInfo?.Author?.Name) return <></>;
        if ([LOG_ACTION.SIGN_IN, LOG_ACTION.SIGN_OUT, LOG_ACTION.REQUEST_PASSWORD_RESET, LOG_ACTION.PASSWORD_CHANGED, LOG_ACTION.SIGN_UP, LOG_ACTION.ESTABLISH_COMPANY, LOG_ACTION.ADD_COMPANY, LOG_ACTION.ACTIVATE_ACCOUNT, LOG_ACTION.UPDATE_USER, LOG_ACTION.REQUEST_INTEGRATION].includes(activity?.LogAction)) {
            if (activity?.LogInfo?.Author?.ID == currentUser?.UserID && activity?.LogInfo?.User?.ID == currentUser?.UserID) return <></>;
        }
        // if ([LOG_ACTION.SIGN_IN, LOG_ACTION.SIGN_OUT, LOG_ACTION.REQUEST_PASSWORD_RESET, LOG_ACTION.PASSWORD_CHANGED, LOG_ACTION.SIGN_UP, LOG_ACTION.ESTABLISH_COMPANY, LOG_ACTION.ADD_COMPANY, LOG_ACTION.ACTIVATE_ACCOUNT].includes(activity?.LogAction)) {
        //     if (activity?.LogInfo?.Author?.ID == currentUser?.UserID) return <></>;
        // }

        if (activity?.LogInfo?.Author?.Status == ITEM_STATUS.PERMANENTLY_DELETED) return <span className="text-muted">, by {activity?.LogInfo?.Author?.Name} (Removed)</span>;
        return <span className="text-muted">, by <Link to={`/users/${activity?.LogInfo?.Author?.ID}`}>{activity?.LogInfo?.Author?.Name}</Link></span>;
    }

    return (
        <tr>
            <td className="d-flex align-items-center">
                <ActivityAvatar
                    activity={activity}
                />
                <div className="align-items-center d-md-flex flex-grow-1 ml-3">
                    <div className="w-75 detail-container">
                        <Description
                            activity={activity}
                            actionItem={actionItem}
                            actionItemType={actionItemType}
                            actionItemData={actionItemData}
                        />
                        {getLogCreatedAt()}{getAuthor()}
                    </div>
                    
                </div>
            </td>
            <td>
                {!!actionItem && <PriorityIndicator priorityType={actionItemData?.PriorityType}/>}
            </td>
            <td>
                {!!actionItem && <SourceIndicator sourceType={actionItemData?.SourceType} />}
            </td>
            <td>
                {
                    actionItem &&
                    <ActionButtons
                        logData={activity}
                        onAddToGroup={onAddToGroup}
                        onAddIntegration={onAddIntegration}
                        actionItem={actionItemData}
                        onRemoveActionItem={handleRemoveActionItem}
                        onImportUsers={onImportUsers}
                        onRestoreUser={onRestoreUser}
                        actionItemData={actionItemData}

                        onRemoveUser={onRemoveUser}
                        onInviteUser={onInviteUser}
                    />
                }
            </td>
        </tr>
    )
}

const mapStateToProps = (state: AppState) => ({
    currentUser: state.Auth?.user,
});

const connector = connect(mapStateToProps, null);

export default connector(ActivityItem);
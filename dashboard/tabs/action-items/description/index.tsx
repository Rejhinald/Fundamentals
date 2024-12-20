import React from 'react';
import { ACTION_ITEM_TYPE, ACTION_PRIORITY } from '../../../../../utils/constants';
import StatusBadgeTooltip from '../../../../groups-info/components/integrations/components/tooltips/status-badge';
import { Badge } from 'react-bootstrap';
import PriorityIndicator from '../../../../activities/components/activity-item/priority-indicator';

const ActionDescription = ({ actionType, actionItemData }) => {
    
    const displayDescription = (actionType) => {
        let description;

        switch (actionType) {
            case ACTION_ITEM_TYPE.CREATE_DEPARTMENT:
                description = "There are no departments yet. Would you like to create your first?";
                break;
            case ACTION_ITEM_TYPE.CREATE_USER:
                description = "There are no users yet. Would you like to create your first?";
                break;
            default:
                break;
        }

        return description;
    }

    return (
        <p className="mb-0">
            {displayDescription(actionType)} {!!actionItemData && <PriorityIndicator priorityType={actionItemData?.PriorityType}/>}
        </p>
    )
}

export default ActionDescription;
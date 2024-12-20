import React, { useEffect } from 'react';
import { ListGroup, OverlayTrigger, Tooltip } from 'react-bootstrap';
import ActionAvatar from '../avatar';
import ActionDescription from '../description';
import ActionButtons from '../buttons';
import dayjs from 'dayjs';
import { AppState } from '../../../../../redux/reducers';
import { connect } from 'react-redux';
import { getCompany } from '../../../../../redux/company/action';
import { deleteActionItem } from '../../../../../redux/action-items/action';
import PriorityIndicator from '../../../../activities/components/activity-item/priority-indicator';
import helper from '../../../../../utils/helper';

const ActionList = ({ 
    actionItem, 
    actionItemType,
    companyData,
    getCompany,
    deleteActionItem,
}) => {

    const now = dayjs();
    useEffect(() => {
        !!actionItem.CompanyID && getCompany({companyId: actionItem.CompanyID})
    },[])

    const handleRemoveActionItem = () => {
        deleteActionItem({actionItemId: actionItem.ActionItemID, companyId: actionItem.CompanyID })
    }
    
    return (
        <ListGroup.Item className="d-flex align-items-center">
            {/* <div className="mr-3 px-0">
                <PriorityIndicator 
                    priorityType={actionItem?.PriorityType}
                />  
            </div> */}
            <ActionAvatar 
                companyData={companyData}
            />
            <div className="align-items-center d-md-flex flex-grow-1 ml-3">
                <div>
                    <ActionDescription 
                        actionType={actionItemType} 
                        actionItemData={actionItem}
                    />
                    <span className="text-muted font-size-sm">{helper.formatDateWithFromNow(actionItem.CreatedAt)}</span>
                </div>
                  
                <ActionButtons 
                    actionType={actionItemType}
                    onRemoveActionItem={handleRemoveActionItem}
                />
                
            </div>
        </ListGroup.Item>
    );
}

const mapStateToProps = (state: AppState) => {
    return {
        companyData: state.Companies.company,
    }
}

const mapDispatchToProps = {
    getCompany,
    deleteActionItem,
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionList);
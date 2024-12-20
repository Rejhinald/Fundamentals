import React, { useState, useEffect } from 'react';
import { Button, Dropdown, OverlayTrigger, Tooltip } from 'react-bootstrap';
import { useHistory } from "react-router-dom";
import { useSelector } from "react-redux";
import { AppState } from "../../../../redux/reducers";
import { ACTION_ITEM_TYPE } from '../../../../utils/constants';
import LoadingButton from '../../../../components/loading-button';
import { IoIosArrowDown } from 'react-icons/io';
import { useCurrentUser } from '../../../../hooks/Auth';

const ActionButtons = ({ actionType, onRemoveActionItem }) => {
    const history = useHistory();

    const currentUser = useSelector((state: AppState) => state.Auth.user);
    const isDeleting = useSelector((state: AppState) => state.ActionItems.isDeleting)

    const [submitting, setSubmitting] = useState(false)

    useEffect(() => {
        if (submitting && !isDeleting) {
            setSubmitting(false)
        }
    }, [isDeleting])

    const handleDismiss = () => {
        setSubmitting(true)
        onRemoveActionItem();
    }

    switch (actionType) {
        case ACTION_ITEM_TYPE.CREATE_DEPARTMENT:
            return (
                <ButtonComponent disabled={!(currentUser.Permissions?.includes('ADD_DEPARTMENT')) || submitting} leftText='Add Now' rightText='Dismiss' pathname='departments' isSubmitting={submitting} onClick={handleDismiss} />
            )
        case ACTION_ITEM_TYPE.CREATE_USER:
            return (
                <ButtonComponent disabled={!(currentUser.Permissions?.includes('ADD_COMPANY_MEMBER')) || submitting} leftText='Add Now' rightText='Dismiss' pathname='users' isSubmitting={submitting} onClick={handleDismiss} />
            )
        default:
            return <></>
    }

}


export default ActionButtons;


interface ButtonComponentProps {
    pathname: string
    disabled: boolean
    isSubmitting: boolean
    leftText: string
    rightText: string
    onClick?: () => void
}
const ButtonComponent = ({ pathname, disabled, isSubmitting, leftText, rightText, onClick }: ButtonComponentProps) => {
    const history = useHistory();

    //?FUNCTION
    const handleNavigation = () => history.push({ pathname })
    const currentUser = useCurrentUser();
    return (
        <div className="ml-md-auto">
              <Dropdown>
                    <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                    <OverlayTrigger trigger={["hover", "focus"]} placement="top" overlay={
                    (<Tooltip id="tooltip-disabled" 
                    style={{display: !currentUser.Permissions || !currentUser?.Permissions?.includes("ADD_COMPANY_MEMBER") ? "block" : "none"}}
                    >You do not have permission to perform any action.</Tooltip>)
                    } >
                        <Button variant="outline-primary" disabled={!currentUser.Permissions || !currentUser?.Permissions?.includes("ADD_COMPANY_MEMBER")}>
                                <span className="pl-2">
                                Actions 
                                <IoIosArrowDown className="ml-5 ml-2" />
                                </span>
                        </Button>
                    </OverlayTrigger>
                    </Dropdown.Toggle>
                    <Dropdown.Menu >
                        <Dropdown.Item className="align-items-center" disabled={disabled} aria-label={`Add now`} onClick={handleNavigation}>
                                <span>{leftText} </span>
                        </Dropdown.Item>
                        <Dropdown.Item className="align-items-center" disabled={disabled}  onClick={onClick && onClick}>
                                <span>{rightText}</span>
                        </Dropdown.Item>
                    </Dropdown.Menu>
                </Dropdown>
        </div>
    )
}
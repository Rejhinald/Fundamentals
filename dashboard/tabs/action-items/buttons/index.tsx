import React, { useState, useEffect } from 'react';
import { Button } from 'react-bootstrap';
import { useHistory } from "react-router-dom";
import { useSelector } from "react-redux";
import { AppState } from "../../../../../redux/reducers";
import { ACTION_ITEM_TYPE } from '../../../../../utils/constants';
import LoadingButton from '../../../../../components/loading-button';

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

    return (
        <div className="ml-md-auto">
            <Button size="sm" className="action-items__btn" disabled={disabled} onClick={handleNavigation}>{leftText}</Button>
            <LoadingButton loading={isSubmitting} disabled={disabled} size="sm" onClick={onClick && onClick} variant="link" className="mx-2">{rightText}</LoadingButton>
        </div>
    )
}
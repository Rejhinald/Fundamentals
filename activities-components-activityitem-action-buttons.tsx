import React, { useState, useEffect } from "react";
import { Badge, Button, Dropdown, OverlayTrigger, Tooltip } from "react-bootstrap";
import { ACTION_ITEM_TYPE, ITEM_STATUS, RESEND_EMAIL_TIMER } from "../../../../../utils/constants";
import { connect } from 'react-redux';
import { AppState } from "../../../../../redux/reducers";
import LoadingButton from "../../../../../components/loading-button";
import "./style.scss";
import { deleteUser } from "../../../../../redux/users/action";
import { getCompanySubscriptionRequest } from "../../../../../redux/subscriptions/action";
import { deleteActionItem } from "../../../../../redux/action-items/action";
import { DELETE_ACTION_ITEM_SUCCESS } from "../../../../../redux/action-items/types";
import Swal from "sweetalert2";
import withReactContent from "sweetalert2-react-content";
import action from "../../../../../redux/auth/action";
import dayjs from "dayjs";
import TimerButton from "../../../../../components/timer-button";
import { PERMISSIONS } from "../../../../../constants/Permissions";
import useCountSeatsAvailable from "../../../../../hooks/useCountSeatsAvailable";
import { PEOPLE_ALERTS } from "../../../../../constants/People";
import { FaCog } from "react-icons/fa";
import { RiPencilFill } from "react-icons/ri";
import { IoIosAdd, IoIosArrowDown } from "react-icons/io";
import { userRequests } from "../../../../../services/request";

interface ActionButtonsProps {
    logData?,
    onAddToGroup?: () => void,
    onAddIntegration?: (filter: string[]) => void,
    onRemoveActionItem: () => void,
    onImportUsers,
    isDeleting?: boolean,
    currentUser?,
    deleteUser?,
    getCompanySubscriptionRequest?,
    actionItem?,
    deleteActionItem?,
    onRestoreUser?: () => void,

    userEvents?,
    actionItemData?,

    onInviteUser?: () => void,
    onRemoveUser?: () => void,
}

export const excludeUsersWithStatusOnRestore = [ITEM_STATUS.PENDING, ITEM_STATUS.ACTIVE, ITEM_STATUS.DEFAULT];

const ActionButtons: React.FC<ActionButtonsProps> = (props) => {
    const {
        logData,
        onAddToGroup,
        onRemoveActionItem,
        onAddIntegration,
        isDeleting,
        onImportUsers,
        currentUser,
        deleteUser,
        deleteActionItem,
        actionItem,
    } = props;

    const { actionItemData } = props;
    const {
        onRemoveUser,
        onInviteUser,
        getCompanySubscriptionRequest
    } = props

    const { onRestoreUser } = props;
    const { userEvents } = props;

    const { seatsAvailable } = useCountSeatsAvailable();
    const hasSeatsAvailable = () => (seatsAvailable > 0);

    const [state, setState] = useState({
        submitting: false,

        submitted: false,

        restoring: false,
    });
    const [userExist, setUserExist] = useState(true)
    //! removed since it is called multiple times depending on how many actionitems there is in the dashboard. 
    //! moved to the Dashboard main component.
    // useEffect(() => {
    //     getCompanySubscriptionRequest({ uid: currentUser?.UserID, cid: currentUser?.ActiveCompany });
    // }, [])

    useEffect(() => {
        if (state.submitting && !isDeleting) {
            setState(s => ({ ...s, submitting: false }));
        }
    }, [isDeleting])

    useEffect(() => {
        if (state?.restoring) {
            if (!userEvents?.restore?.loading && userEvents?.restore?.success) {
                setState(s => ({ ...s, restoring: false }));
            }
            getCompanySubscriptionRequest({ uid: currentUser?.UserID, cid: currentUser?.ActiveCompany });
        }
    }, [userEvents]);

    useEffect(() => {
        if (actionItem?.ActionItemType == ACTION_ITEM_TYPE.INVITE_REMOVE_COMPANY_MEMBER) {
            const userParams = {
                include: "roles",
                skipLoading: true
            }
            const user = userRequests.getUser(actionItem?.Log?.UserID, userParams).then(a => {
                if (a.data.status.code !== "200") {
                    setUserExist(false)
                } 
            })
        }
    },[])


    const handleDismiss = () => {
        setState(s => ({ ...s, submitting: true }));
        onRemoveActionItem();
    }

    const handleRemoveUser = (actionItemArg) => {
        const { ID, Name } = actionItemArg.Log.LogInfo.Users[0]

        const MySwal = withReactContent(Swal)

        MySwal.fire({
            icon: "warning",
            title: `Remove ${Name}`,
            text: `Are you sure you want to remove ${Name}?`,
            showCancelButton: true,
            heightAuto: false,
            confirmButtonText: "Remove",
            customClass: {
                confirmButton: "btn btn-danger order-2",
                cancelButton: "btn btn-link text-muted"
            },
            buttonsStyling: false
        }).then(result => {
            if (result.isConfirmed) {
                deleteUser({
                    userId: ID,
                    companyId: currentUser?.ActiveCompany,
                })

                deleteActionItem({ actionItemId: actionItemArg.ActionItemID })
            }
        });
    }

    const handleRestoreUser = () => {
        setState(s => ({ ...s, restoring: true }));
        onRestoreUser && onRestoreUser();
    }

    // const excludeUsersWithStatusOnRestore = [ITEM_STATUS.PENDING, ITEM_STATUS.ACTIVE, ITEM_STATUS.DEFAULT];

    // 
    const getButton = () => {
        if (!currentUser.Permissions) {
            return <></>
        }
        switch (actionItem?.ActionItemType) {
            case ACTION_ITEM_TYPE.ADD_USERS_TO_GROUP: {
                if (logData.LogInfo?.Group?.Status === ITEM_STATUS.INACTIVE || logData.LogInfo?.Group?.Status === ITEM_STATUS.DELETED) {
                    return <></>
                } else {
                    return (
                        (currentUser.Permissions?.includes("ADD_GROUP_MEMBER")) ?
                            (

                                <div className="ml-md-auto action_item__buttons">
                                    <Dropdown>
                                        <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                            <Button variant="outline-primary">
                                                <span className="pl-2">
                                                    Actions
                                                    <IoIosArrowDown className="ml-5 ml-2" />
                                                </span>
                                            </Button>
                                        </Dropdown.Toggle>
                                        <Dropdown.Menu>
                                            <Dropdown.Item className="align-items-center" aria-label={`Add now`} onClick={onAddToGroup}>
                                                <span>Add Now </span>
                                            </Dropdown.Item>
                                            <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                                <span>Dismiss</span>
                                            </Dropdown.Item>
                                        </Dropdown.Menu>
                                    </Dropdown>
                                </div>
                            )

                            :

                            (
                                <div className="ml-md-auto action_item__buttons">
                                    <Dropdown>
                                        <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                            {/* <OverlayTrigger trigger="hover" placement="top" overlay={<Tooltip id="tooltip-disabled">You do not have permission to perform any action.</Tooltip>}> */}
                                            <Button variant="outline-primary" disabled={!currentUser.Permissions || !currentUser?.Permissions?.includes("ADD_GROUP_MEMBER")}>
                                                <span className="pl-2">
                                                    Actions
                                                    <IoIosArrowDown className="ml-5 ml-2" />
                                                </span>
                                            </Button>
                                            {/* </OverlayTrigger> */}
                                        </Dropdown.Toggle>
                                        <Dropdown.Menu>

                                            <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                                <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Add now`} style={{ pointerEvents: 'none' }} onClick={onAddToGroup}>
                                                    <span>Add Now</span>
                                                </Dropdown.Item>
                                            </OverlayTrigger>
                                            <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                                <span>Dismiss</span>
                                            </Dropdown.Item>
                                        </Dropdown.Menu>
                                    </Dropdown>
                                </div>

                            )
                    )
                }

            }
            case ACTION_ITEM_TYPE.ADD_REMOVE_USER: {
                const isCurrentUser: boolean = (actionItem?.Log?.LogInfo?.Users?.map(user => user?.ID).includes(currentUser?.UserID)) || false
                return (
                    <div className="ml-md-auto action_item__buttons">
                        <Dropdown>
                            <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                {/* <OverlayTrigger trigger="hover" placement="top" overlay={<Tooltip id="tooltip-disabled">You do not have permission to perform any action.</Tooltip>}> */}
                                <Button variant="outline-primary" disabled={!currentUser.Permissions || !currentUser?.Permissions?.includes("REMOVE_COMPANY_MEMBER")}>
                                    <span className="pl-2">
                                        Actions
                                        <IoIosArrowDown className="ml-5 ml-2" />
                                    </span>
                                </Button>
                                {/* </OverlayTrigger> */}
                            </Dropdown.Toggle>
                            <Dropdown.Menu>
                                {
                                    !!(currentUser.Permissions?.includes("ADD_GROUP_MEMBER")) ? (

                                        <Dropdown.Item className="align-items-center action-items__btn" aria-label={`Add now`} onClick={onAddToGroup}>
                                            <span>Add Now </span>
                                        </Dropdown.Item>
                                    )
                                        :
                                        (<OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                            <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Add now`} style={{ pointerEvents: 'none' }} onClick={onAddToGroup}>
                                                <span>Add Now</span>
                                            </Dropdown.Item>
                                            {/* <span><Button size="sm" style={{ pointerEvents: 'none' }} className="opacity-40 action-items__btn  " onClick={onAddToGroup}>Add Now</Button></span> */}
                                        </OverlayTrigger>)
                                }

                                {
                                    !!(currentUser.Permissions?.includes("REMOVE_GROUP_MEMBER")) ? (

                                        <Dropdown.Item className="align-items-center action-items__btn" aria-label={`Remove user now`} onClick={() => handleRemoveUser(actionItem)}>
                                            <span>Remove User</span>
                                        </Dropdown.Item>
                                        // <Button size="sm" variant="danger" disabled={isCurrentUser} onClick={() => handleRemoveUser(actionItem)}></Button>
                                    )
                                        :
                                        (<OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                            <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Remove user now`} style={{ pointerEvents: 'none' }} onClick={() => handleRemoveUser(actionItem)}>
                                                <span>Add Now</span>
                                            </Dropdown.Item>
                                            {/* <span><Button size="sm" variant="danger" style={{ pointerEvents: 'none' }} className="opacity-40 action-items__btn" onClick={() => handleRemoveUser(actionItem)}>Remove User</Button></span> */}
                                        </OverlayTrigger>)
                                }

                                {
                                    !!((currentUser.Permissions?.includes("ADD_GROUP_MEMBER")) || (currentUser.Permissions?.includes("REMOVE_GROUP_MEMBER"))) && (
                                        <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                            <span>Dismiss</span>
                                        </Dropdown.Item>
                                    )
                                }
                            </Dropdown.Menu>
                        </Dropdown>

                    </div>
                )
            }

            case ACTION_ITEM_TYPE.SUGGEST_INTEGRATION_CATEGORY:
            case ACTION_ITEM_TYPE.SUGGEST_HOT_INTEGRATION:
                return (
                    (currentUser.Permissions?.includes("CONNECT_INTEGRATION")) ?
                        (
                            <div className="ml-md-auto action_item__buttons">
                                <Dropdown>
                                    <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                        <Button variant="outline-primary">
                                            <span className="pl-2">
                                                Actions
                                                <IoIosArrowDown className="ml-5 ml-2" />
                                            </span>
                                        </Button>
                                    </Dropdown.Toggle>
                                    <Dropdown.Menu>
                                        <Dropdown.Item className="align-items-center" aria-label={`Add now`} onClick={() => onAddIntegration && onAddIntegration(actionItem?.Log?.LogInfo?.Integration.Categories)}>
                                            <span>Add Now </span>
                                        </Dropdown.Item>
                                        <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                            <span>Dismiss</span>
                                        </Dropdown.Item>
                                    </Dropdown.Menu>
                                </Dropdown>
                            </div>
                        )
                        :
                        (
                            <div className="ml-md-auto action_item__buttons">
                                <Dropdown>
                                    <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                        {/* <OverlayTrigger trigger="hover" placement="top" overlay={<Tooltip id="tooltip-disabled">You do not have permission to perform any action.</Tooltip>}> */}
                                        <Button variant="outline-primary" disabled={!currentUser.Permissions || !currentUser?.Permissions?.includes("CONNECT_INTEGRATION")}>
                                            <span className="pl-2">
                                                Actions
                                                <IoIosArrowDown className="ml-5 ml-2" />
                                            </span>
                                        </Button>
                                        {/* </OverlayTrigger> */}
                                    </Dropdown.Toggle>
                                    <Dropdown.Menu>
                                        <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                            <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Add now`} style={{ pointerEvents: 'none' }} onClick={() => onAddIntegration && onAddIntegration(actionItem?.Log?.LogInfo?.Integration.Categories)}>
                                                <span>Add Now</span>
                                            </Dropdown.Item>
                                        </OverlayTrigger>
                                        {/* <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                            <span>Dismiss</span>
                                        </Dropdown.Item> */}
                                    </Dropdown.Menu>
                                </Dropdown>
                            </div>
                        )
                    // !!(currentUser.Permissions?.includes("CONNECT_INTEGRATION")) && (
                    //     <div className="ml-md-auto action_item__buttons">
                    //         <Button size="sm" className="action-items__btn" onClick={() => onAddIntegration && onAddIntegration(actionItemData?.Log?.LogInfo?.Integration?.Categories)}>Add Now</Button>
                    //         <LoadingButton loading={state.submitting} size="sm" onClick={handleDismiss} variant="link" className="">Dismiss</LoadingButton>
                    //     </div>
                    // )
                )
            case ACTION_ITEM_TYPE.IMPORT_INTEGRATION_CONTACTS:
                return (
                    (currentUser.Permissions?.includes("CONNECT_INTEGRATION")) ?
                        (
                            <div className="ml-md-auto action_item__buttons">
                                <Dropdown>
                                    <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                        <Button variant="outline-primary">
                                            <span className="pl-2">
                                                Actions
                                                <IoIosArrowDown className="ml-5 ml-2" />
                                            </span>
                                        </Button>
                                    </Dropdown.Toggle>
                                    <Dropdown.Menu>
                                        <Dropdown.Item className="align-items-center" aria-label={`Import now`} onClick={onAddToGroup}>
                                            <span>Import</span>
                                        </Dropdown.Item>
                                        <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                            <span>Dismiss</span>
                                        </Dropdown.Item>
                                    </Dropdown.Menu>
                                </Dropdown>
                            </div>
                        )
                        :
                        (
                            <div className="ml-md-auto action_item__buttons">
                                <Dropdown>
                                    <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                        {/* <OverlayTrigger trigger="hover" placement="top" overlay={<Tooltip id="tooltip-disabled">You do not have permission to perform any action.</Tooltip>}> */}
                                        <Button variant="outline-primary" disabled={!currentUser.Permissions || !currentUser?.Permissions?.includes("CONNECT_INTEGRATION")}>
                                            <span className="pl-2">
                                                Actions
                                                <IoIosArrowDown className="ml-5 ml-2" />
                                            </span>
                                        </Button>
                                        {/* </OverlayTrigger> */}
                                    </Dropdown.Toggle>
                                    <Dropdown.Menu>
                                        <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                            <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Import now`} onClick={onAddToGroup} style={{ pointerEvents: 'none' }} >
                                                <span>Import</span>
                                            </Dropdown.Item>
                                        </OverlayTrigger>
                                        <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                            <span>Dismiss</span>
                                        </Dropdown.Item>
                                    </Dropdown.Menu>
                                </Dropdown>
                            </div>
                        )
                    // !!(currentUser.Permissions?.includes("CONNECT_INTEGRATION")) && (
                    //     <div className="ml-md-auto action_item__buttons">
                    //         <Button size="sm" className="action-items__btn" onClick={onImportUsers}>Import</Button>
                    //         <LoadingButton loading={state.submitting} size="sm" onClick={handleDismiss} variant="link" className="">Dismiss</LoadingButton>
                    //     </div>
                    // )
                )
            case ACTION_ITEM_TYPE.REMOVE_USER_TO_COMPANY:
                return (
                    <div className="ml-md-auto action_item__buttons">
                        <Dropdown>
                            <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                {/* <OverlayTrigger trigger="hover" placement="top" overlay={<Tooltip id="tooltip-disabled">You do not have permission to perform any action.</Tooltip>}> */}
                                <Button variant="outline-primary" disabled={!currentUser.Permissions || !currentUser?.Permissions?.includes("REMOVE_COMPANY_MEMBER")}>
                                    <span className="pl-2">
                                        Actions
                                        <IoIosArrowDown className="ml-5 ml-2" />
                                    </span>
                                </Button>
                                {/* </OverlayTrigger> */}
                            </Dropdown.Toggle>
                            <Dropdown.Menu>
                                {
                                    actionItem?.Log?.LogInfo?.Users?.[0]?.Temp?.Status ? (
                                        <>
                                            {
                                                excludeUsersWithStatusOnRestore?.includes(actionItem?.Log?.LogInfo?.Users?.[0]?.Temp?.Status) ? (
                                                    <Dropdown.Item disabled>
                                                        {/* <Badge className="bg-light">Restoredd</Badge> */}
                                                        Restored
                                                    </Dropdown.Item>
                                                ) : (
                                                    <>
                                                        <>
                                                            {
                                                                !hasSeatsAvailable() ?
                                                                    (
                                                                        currentUser.Permissions?.includes('ADD_COMPANY_MEMBER') ?
                                                                            (
                                                                                <OverlayTrigger trigger={["hover", "focus"]} overlay={<Tooltip id="tooltip-disabled">{PEOPLE_ALERTS.ADD_USER.FAILED_LICENSE}</Tooltip>}>
                                                                                    <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Restor now`} style={{ pointerEvents: 'none' }} onClick={onAddToGroup}>

                                                                                        <span className="d-inline-block">
                                                                                            {/* <Button size="sm" disabled style={{ pointerEvents: 'none' }}> */}
                                                                                            Restore
                                                                                            {/* </Button> */}
                                                                                        </span>
                                                                                    </Dropdown.Item>
                                                                                </OverlayTrigger >
                                                                            )
                                                                            :
                                                                            (
                                                                                <div className="ml-md-auto action_item__buttons">

                                                                                    <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                                                                        <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Restore now`} style={{ pointerEvents: 'none' }} onClick={handleRestoreUser}>

                                                                                            <span>
                                                                                                {/* <Button size="sm" disabled style={{ pointerEvents: 'none' }}> */}
                                                                                                Restore
                                                                                                {/* </Button> */}
                                                                                            </span>
                                                                                        </Dropdown.Item>
                                                                                    </OverlayTrigger>
                                                                                    <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                                                                        <span>Dismiss</span>
                                                                                    </Dropdown.Item>
                                                                                </div>
                                                                            )
                                                                    ) : (
                                                                        currentUser.Permissions?.includes('ADD_COMPANY_MEMBER') ?
                                                                            (
                                                                                <Dropdown.Item>
                                                                                    <LoadingButton variant={'link'} disabled={!hasSeatsAvailable()} loading={userEvents?.restore?.loading && state.restoring} className="action-items__btn text-left p-0" onClick={handleRestoreUser}>Restore</LoadingButton>
                                                                                </Dropdown.Item>
                                                                            )
                                                                            :
                                                                            (
                                                                                <div className="ml-md-auto action_item__buttons">
                                                                                    <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                                                                        <Dropdown.Item className="align-items-center opacity-40 action-items__btn" aria-label={`Restor now`} style={{ pointerEvents: 'none' }} onClick={handleRestoreUser}>
                                                                                            <span> <Button size="sm" style={{ pointerEvents: 'none' }} className="opacity-40 action-items__btn" onClick={handleRestoreUser}>Restore</Button></span>
                                                                                        </Dropdown.Item>
                                                                                    </OverlayTrigger>
                                                                                    <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                                                                        <span>Dismiss</span>
                                                                                    </Dropdown.Item>
                                                                                </div>
                                                                            )
                                                                    )
                                                            }

                                                        </>
                                                    </>

                                                )
                                            }
                                        </>
                                    ) : (
                                        <Dropdown.Item disabled>
                                            {/* <Badge className="bg-light">Restored</Badge> */}
                                            Restored
                                        </Dropdown.Item>
                                    )
                                }
                                {
                                    currentUser.Permissions?.includes('ADD_COMPANY_MEMBER') ? (
                                        <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                            <span>Dismiss</span>
                                        </Dropdown.Item>
                                    ) : <></>
                                }
                            </Dropdown.Menu>
                        </Dropdown>

                    </div >
                )
            case ACTION_ITEM_TYPE.INVITE_REMOVE_COMPANY_MEMBER: {
                const isCurrentUser: boolean = (actionItem?.Log?.UserID === currentUser?.UserID) || false
                const date1 = dayjs.unix(actionItemData?.Extras?.CreatedAt);
                const date2 = dayjs.unix(actionItemData?.CreatedAt);
                const hours = date2.diff(date1, "hours");
                const days = Math.floor(hours / 24);
                return (
                    <div className="ml-md-auto action_item__buttons">
                        <Dropdown>
                            <Dropdown.Toggle variant="link" id="dropdown-menu" className="cog-icon no-caret">
                                {/* <OverlayTrigger trigger="hover" placement="top" overlay={<Tooltip id="tooltip-disabled">You do not have permission to perform any action.</Tooltip>}> */}
                                <Button variant="outline-primary" disabled={!currentUser.Permissions || !currentUser?.Permissions?.includes("ADD_COMPANY_MEMBER")}>
                                    <span className="pl-2">
                                        Actions
                                        <IoIosArrowDown className="ml-5 ml-2" />
                                    </span>
                                </Button>
                                {/* </OverlayTrigger> */}
                            </Dropdown.Toggle>
                            <Dropdown.Menu>
                                {
                                    (actionItemData?.Log?.LogInfo?.User?.Status === ITEM_STATUS.PENDING) ? (
                                        <>
                                            {
                                                !!currentUser?.Permissions?.includes(PERMISSIONS.ADD_COMPANY_MEMBER) ? (
                                                    userExist ?
                                                    <Dropdown.Item className="align-items-center" aria-label={`Resend Invite`}>
                                                        <TimerButton className="p-0 btn btn-lg" removeVariant={true} duration={RESEND_EMAIL_TIMER} onClick={onInviteUser}>
                                                            Resend Invite
                                                        </TimerButton>
                                                    </Dropdown.Item>
                                                    :
                                                    <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">User is deleted. Restore the user</Tooltip>}>
                                                        <Dropdown.Item className="align-items-center" aria-label={`Resend Invite`} disabled>
                                                            <TimerButton className="p-0 btn btn-lg" removeVariant={true} duration={RESEND_EMAIL_TIMER} onClick={onInviteUser}>
                                                                Resend Invite
                                                            </TimerButton>
                                                        </Dropdown.Item>
                                                    </OverlayTrigger>
                                                ) :
                                                    (
                                                        <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                                            <Dropdown.Item style={{ pointerEvents: 'none' }} className="align-items-center opacity-40 action-items__btn" aria-label={`Resend Invite`}>
                                                                <span>
                                                                    Resend Invite
                                                                    {/* <TimerButton duration={RESEND_EMAIL_TIMER} onClick={onInviteUser} size="sm" style={{ pointerEvents: 'none' }} className="opacity-40 action-items__btn">Resend Invite</TimerButton> */}
                                                                </span>
                                                            </Dropdown.Item>
                                                        </OverlayTrigger>
                                                    )
                                            }
                                            {/* <LoadingButton size="sm" onClick={onInviteUser}></LoadingButton> */}
                                            {
                                                days > 3 ? (
                                                    currentUser?.Permissions?.includes(PERMISSIONS.REMOVE_COMPANY_MEMBER) ? (
                                                        userExist ?
                                                        <Dropdown.Item className="align-items-center" variant="primary" disabled={isCurrentUser} onClick={onRemoveUser}>
                                                            
                                                            <span className="font-size-lg">Remove User</span>
                                                        </Dropdown.Item>
                                                        :
                                                        <Dropdown.Item className="align-items-center" disabled={isCurrentUser} onClick={handleRestoreUser}>
                                                        
                                                            <span className="font-size-lg">Restore User</span>
                                                    </Dropdown.Item>
                                                        // <LoadingButton size="sm" variant="danger" className="" disabled={isCurrentUser} onClick={onRemoveUser}>Remove User</LoadingButton>
                                                    ) : (
                                                        userExist ?
                                                        <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                                            <Dropdown.Item style={{ pointerEvents: 'none' }} className="align-items-center opacity-40 action-items__btn" aria-label={`Resend Invite`} onClick={onRemoveUser} disabled>
                                                                <span>
                                                                    {/* <Button size="sm" variant="danger" style={{ pointerEvents: 'none' }} className="opacity-40 action-items__btn "> */}
                                                                    Remove User
                                                                    {/* </Button> */}
                                                                </span>
                                                            </Dropdown.Item>
                                                        </OverlayTrigger>
                                                        :
                                                        <OverlayTrigger trigger={["hover", "focus"]} placement="left" overlay={<Tooltip id="tooltip-disabled">You do not have permission to do this action.</Tooltip>}>
                                                            <Dropdown.Item style={{ pointerEvents: 'none' }} className="align-items-center opacity-40 action-items__btn" aria-label={`Resend Invite`} onClick={onRemoveUser} disabled>
                                                                <span>
                                                                    {/* <Button size="sm" variant="danger" style={{ pointerEvents: 'none' }} className="opacity-40 action-items__btn "> */}
                                                                    Restore User
                                                                    {/* </Button> */}
                                                                </span>
                                                            </Dropdown.Item>
                                                        </OverlayTrigger>
                                                    )
                                                ) : (
                                                    <></>
                                                )
                                            }
                                        </>
                                    ) : (
                                        // <Badge>Active</Badge>
                                        <></>
                                    )
                                }
                                {
                                    currentUser?.Permissions?.includes(PERMISSIONS.ADD_COMPANY_MEMBER) && currentUser?.Permissions?.includes(PERMISSIONS.REMOVE_COMPANY_MEMBER) ? (
                                        <Dropdown.Item className="align-items-center" onClick={handleDismiss}>
                                            <span>Dismiss</span>
                                        </Dropdown.Item>
                                    ) : <></>
                                }
                            </Dropdown.Menu>
                        </Dropdown>
                    </div>
                )
                               
            }

            default:
                return <></>
        }
        return <></>
    }

    return (
        <>
            {getButton()}
        </>
    )
}

const mapDispatchToProps = {
    deleteUser,
    deleteActionItem,
    getCompanySubscriptionRequest,
}

const mapStateToProps = (state: AppState) => {
    return {
        isDeleting: state.ActionItems.isDeleting,
        currentUser: state.Auth.user,
        userEvents: state.Users?.events || {},
    }
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionButtons);
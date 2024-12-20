import React, { useEffect, useState } from "react";
import {
    Button,
    Card,
    ListGroup,
    Image
} from "react-bootstrap";
import { BsPlus } from "react-icons/bs";
import { useHistory } from "react-router-dom";
import { Link } from "react-router-dom";
import dayjs from "dayjs";

import { AppState } from "../../../../redux/reducers";
import { sortActionItems } from "../../../../redux/action-items/action";
import { connect, ConnectedProps, useDispatch } from 'react-redux';
import { resendActivation } from "../../../../redux/auth/action";

import ActivityItem from "../../../activities/components/activity-item";
import EmptyStateLoading from "../../components/empty-state";
import PageFilter from "../../../../components/page-filter";
import EmptyState from "../../../../components/empty-state";
import _ from 'lodash';

import { RemoveCompanyUserParams } from "../../../../interfaces/forms/User";
import noActionItemsIcon from "../../../../assets/images/empty_state_action_item.svg";
import resultNotFoundIcon from "../../../../assets/images/result_not_found.svg";


import { actionItemsSelector, getSortedActionItems, sortSelector } from "../../selectors";

import ActionList from "./list-group";
import { setSelectedIds } from "../../../../redux/local-states/people-module/action";
import { removeCompanyUserConfirmation } from "../../../users/components/confirmation";
import { ACTION_PRIORITY } from "../../../../utils/constants";
import LoadingState from "../../components/loading-state";


interface ActionItemsProps {
    actionItems,
    users,
    departments,
    groups,
    onRemoveActionItem: (actionItemId: string) => void,
    onRestoreUser: (userId: string) => void,
}
//: React.FC<ActionItemsProps & PropsFromRedux>
const ActionItems: React.FC<ActionItemsProps & PropsFromRedux> = (props) => {


    const dispatch = useDispatch()


    const { actionItems, users, departments, groups, isLoading, currentUser } = props;
    const tmpActionItems = actionItemsSelector;
    const { onRemoveActionItem, sortActionItems } = props;
    const { onRestoreUser } = props;
    const { resendActivation } = props;
    const [searchKey, setSearchKey] = useState<string>("")
    const [priorityType, setPriorityType] = useState<string | undefined>(undefined)
    const [startDate, setStartDate] = useState<Date | null>(null);
    const [endDate, setEndDate] = useState<Date | null>(null);
    const [sortList, setSortList] = useState(false)
    const [activityItemsList, setActivityItemsList] = useState<any>([])
    const history = useHistory();
    const [isSearching, setIsSearching] = useState<boolean>(false);
    const [isFilteringPriority, setIsFilteringPriority] = useState<boolean>(false)
    const [isSorting, setIsSorting] = useState<boolean>(false);

    const handleAddToGroup = (item) => {
        const people = item?.Log?.LogInfo?.Users?.map(user => user?.ID) || [];
        const group = item?.Log?.LogInfo?.Group?.ID || ""

        const pathname = `/users`
        const search = `?add-to-group=true&people=${JSON.stringify(people)}&group=${JSON.stringify(group)}`

        history.push({ pathname, search })
    }

    const handleAddIntegration = (filter: string[]) => {
        history.push({
            pathname: "/integrations",
            search: `${filter?.length == 1 ? `?filter=${filter?.[0]}` : ``}`,
        })
    }

    const handleImportIntegrationUsers = (item) => {
        history.push({
            pathname: "/users",
            search: `? action =import& integration=${item?.Log?.LogInfo?.Integration?.Name} `, // todo change to ID or slug
        })
        
    }

    const handleInviteUser = (userId: string) => {
        resendActivation({ userId });
    }

    const handleRemoveUser = (params: RemoveCompanyUserParams) => {
            
        removeCompanyUserConfirmation(params).then(result => {
            // 
            
        }).catch(error => {
            
        })
    }
   
    const displayActionItems = async () => {
        let lists: JSX.Element[] = [];
        let actionList = actionItems;
        actionList = Object.keys(actionList).sort((a, b) => sortList ? new Date(a).getTime() - new Date(b).getTime() : new Date(b).getTime() - new Date(a).getTime())
            .reduce((result, key) => {
                result[key] = actionList[key];
                return result;
            }, {});

        for (let key in actionList) {
            const filteredActions = actionList[key].filter((res) => {
                return priorityType === undefined || res.PriorityType === priorityType;
            });
            if (searchKey) {
                let activities: JSX.Element[] = [];
                filteredActions.map((item, idx) => {
                    if (item?.Log?.SearchKey) {
                        if (item?.Log?.SearchKey?.includes(searchKey) || item?.Log?.SearchKey?.toLowerCase().includes(searchKey)) {
                            activities.push(
                                <ActivityItem
                                    key={`${key} -${idx} `}
                                    activity={item?.Log}
                                    actionItem={item?.ActionItemID}
                                    actionItemType={item?.ActionItemType}
                                    actionItemData={item}
                                    onAddToGroup={() => handleAddToGroup(item)}
                                    onRemoveActionItem={(id) => onRemoveActionItem(id)}
                                    onImportUsers={() => handleImportIntegrationUsers(item)}
                                    onRestoreUser={() => onRestoreUser(item?.Log?.LogInfo?.Users?.[0]?.Temp?.ID || "")}
                                />
                            )
                        }
                    }

                })
                // if (activities.length) {
                //     activities.unshift(<ListGroup.Item key={key}><h4 className="font-weight-bolder mb-0">{dayjs(key).format("MMMM D")}</h4></ListGroup.Item>);
                //     lists.push(...activities);
                // }

            } else {
                if (filteredActions.length !== 0) {
                    // lists.push(<ListGroup.Item key={key}><h4 className="font-weight-bolder mb-0">{dayjs(key).format("MMMM D")}</h4></ListGroup.Item>);
                    // actionList[key].map((item, idx) => lists.push(
                    //     <ActivityItem
                    //         key={`${ key } -${ idx } `}
                    //         activity={item?.Log}
                    //         actionItem={item?.ActionItemID}
                    //         onAddToGroup={() => handleAddToGroup(item)}
                    //         onRemoveActionItem={(id) => onRemoveActionItem(id)}
                    //     />
                    // ))
                    filteredActions.map((item, idx) => {
                        if (Object.keys(item?.Log || {})?.length > 0) {
                            lists.push(
                                <ActivityItem
                                    key={`${key} -${idx} `}
                                    activity={item?.Log}
                                    actionItem={item?.ActionItemID}
                                    actionItemType={item?.ActionItemType}
                                    actionItemData={item}
                                    onAddToGroup={() => handleAddToGroup(item)}
                                    onAddIntegration={handleAddIntegration}
                                    onRemoveActionItem={(id) => onRemoveActionItem(id)}
                                    onImportUsers={() => handleImportIntegrationUsers(item)}
                                    onRestoreUser={() => onRestoreUser(item?.Log?.LogInfo?.Users?.[0]?.Temp?.ID || "")}

                                    onInviteUser={() => handleInviteUser(item?.Log?.LogInfo?.User?.ID || "")}
                                    onRemoveUser={() => { handleRemoveUser({ userId: item?.Log?.LogInfo?.User?.ID, companyId: item?.CompanyID, name: item?.Log?.LogInfo?.User?.Name, actionItemID: item.ActionItemID }) }}
                                />
                            )
                        } else {
                            //
                            lists.push(
                                <ActionList
                                    key={`${key} -${idx} `}
                                    actionItem={item}
                                    actionItemType={item?.ActionItemType}
                                />
                            )
                        }
                    })
                }

            }
        }

        setIsSorting(false)

        // return lists;
        setActivityItemsList(lists);
    }

    useEffect(() => {
        sortActionItems({ startDate: 0, endDate: 0 });
    }, [])

    useEffect(() => {
        setTimeout(() => {
            // setIsSorting(false)
            displayActionItems()
        }, 500)

    }, [searchKey, sortList, actionItems, isSearching, priorityType, isSorting])

    const handlePriorityType = (value) => {
        setIsSorting(true)
        setPriorityType(value)
        setIsFilteringPriority(!!value)
    }

    const handleDateSort = (sort) => {
        const { startDate, endDate } = sort;
        setStartDate(startDate);
        setEndDate(startDate);
        setIsSorting(true)
        sortActionItems({ startDate: startDate && dayjs(startDate).unix(), endDate: endDate && dayjs(endDate).unix() });
        setIsSearching(!!sort)
        setIsFilteringPriority(!!sort)
    };

    const handleSortList = _.debounce((value) => {
        setIsSorting(true);
        setIsSearching(!!value)
        setIsFilteringPriority(!!value)
        setSortList(value);
    }, 500);

    const handleSearch = _.debounce((key) => {
        setIsSorting(true)
        setSearchKey(key)
        setIsSearching(!!key)
        setIsFilteringPriority(!!key)
    }, 500);



    return (
        <>
            {
                isLoading ?
                    <EmptyStateLoading />
                    :
                    (
                        <>
                            {
                                ((activityItemsList?.length > 0 || isSearching || isFilteringPriority || startDate || endDate || priorityType)) ?
                                    (
                                        <PageFilter
                                            module="dashboard"
                                            onSearch={(key) => handleSearch(key)}
                                            onSort={(sort) => handleDateSort(sort)}
                                            handleSort={(sort) => handleSortList(sort)}
                                            handleSortByPriority={(value) => handlePriorityType(value)}
                                            onFilter={() => { }}
                                            isSearching={isSearching}
                                            isDisabled={!activityItemsList?.length && searchKey != ''}
                                            items={activityItemsList}
                                            setSortLoading={(value) => setIsSorting(value)}

                                            sDate={startDate}
                                            eDate={endDate}
                                            pType={priorityType}
                                        />
                                    ) : <></>
                            }
                            {
                                (isSorting) ?
                                    <LoadingState /> :
                                    (
                                        activityItemsList?.length !== 0 ?
                                            (
                                                <Card className="card-custom flex-grow-1 overflow-auto custom-scrollbar">
                                                    <ListGroup variant="flush">
                                                        {activityItemsList}
                                                    </ListGroup>
                                                </Card>
                                            )
                                            :
                                            (
                                                ((users?.length === 0 || departments?.length === 0 || groups?.length === 0)) && !isSearching ?
                                                    <Card className="card-custom h-100">
                                                        <Card.Body>
                                                            <div className="align-items-center d-flex flex-column h-100 justify-content-center">
                                                                <EmptyState
                                                                    media={<Image src={noActionItemsIcon} alt="no action item" fluid />}
                                                                    content={<p>You don't have an action item at the moment.</p>}
                                                                />
                                                            </div>
                                                        </Card.Body>
                                                    </Card>
                                                    :
                                                    ((users?.length === 0 || departments?.length === 0 || groups?.length === 0) || isSearching || isFilteringPriority) ?
                                                        <Card className="card-custom h-100">
                                                            <Card.Body>
                                                                <div className="align-items-center d-flex flex-column h-100 justify-content-center">
                                                                    <EmptyState
                                                                        title="Sorry, no results found"
                                                                        media={<Image src={resultNotFoundIcon} alt="no result found" fluid />}
                                                                        content={<p>Try adjusting your search to find what you're looking for.</p>}
                                                                    />
                                                                </div>
                                                            </Card.Body>
                                                        </Card>
                                                        :
                                                        (
                                                            <Card className="card-custom h-100">
                                                                <Card.Body>
                                                                    <div className="align-items-center d-flex flex-column h-100 justify-content-center">
                                                                        {

                                                                            users?.length === 0 ?
                                                                                currentUser.Permissions?.includes('ADD_COMPANY_MEMBER') ?
                                                                                    <>
                                                                                        <p className="text-muted">You don't have a user added. Click the "Add People" button to add your first user.</p>
                                                                                        <Link to="/users">
                                                                                            <Button><BsPlus /> Add People</Button>
                                                                                        </Link>
                                                                                    </>
                                                                                    :
                                                                                    <></>
                                                                                :
                                                                                departments?.length === 0 ?
                                                                                    currentUser?.Permissions?.includes('ADD_DEPARTMENT') ?
                                                                                        <>
                                                                                            <p className="text-muted">You don't have any Users yet. Click the "Add People" button to add your first user.</p>
                                                                                            <Link to="/users">
                                                                                                <Button><BsPlus /> Add People</Button>
                                                                                            </Link>
                                                                                        </> : <></>
                                                                                    :
                                                                                    <EmptyState
                                                                                        media={<Image src={noActionItemsIcon} alt="no action item" fluid />}
                                                                                        content={<p>You don't have an action item at the moment.</p>}
                                                                                    />
                                                                        }
                                                                    </div>
                                                                </Card.Body>
                                                            </Card>
                                                        )
                                            )
                                    )
                            }
                        </>
                    )
            }
        </>
    )
}
const mapDispatchToProps = {
    sortActionItems,
    resendActivation
};

const mapStateToProps = (state: AppState) => {
    return {
        //actionItems: getSortedActionItems(state),
        sortParams: sortSelector(state),
        isLoading: state.ActionItems.isLoading,
        currentUser: state.Auth.user,
    }
}

const connector = connect(mapStateToProps, mapDispatchToProps)

type PropsFromRedux = ConnectedProps<typeof connector>

export default connector(ActionItems);
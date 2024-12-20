import React, { useEffect, useState } from "react";
import {
    Button,
    Card,
    ListGroup,
    Image,
    Table,
    Modal
} from "react-bootstrap";
import { BsPlus } from "react-icons/bs";
import { useHistory } from "react-router-dom";
import { Link } from "react-router-dom";
import dayjs from "dayjs";

import { AppState } from "../../../redux/reducers";
import { deleteActionItem,filterActionItems, getActionItems, sortActionItems } from "../../../redux/action-items/action";
import { connect, ConnectedProps, useDispatch } from 'react-redux';
import { resendActivation } from "../../../redux/auth/action";

import ActivityItem from "../action-items/list-action-items";
import EmptyStateLoading from "../components/empty-state";
import PageFilter from "../../../components/page-filter";
import EmptyState from "../../../components/empty-state";
import _ from 'lodash';

import { RemoveCompanyUserParams } from "../../../interfaces/forms/User";
import noActionItemsIcon from "../../../assets/images/empty_state_action_item.svg";
import resultNotFoundIcon from "../../../assets/images/result_not_found.svg";


import { actionItemsSelector, getSortedActionItems, sortSelector } from "../selectors";

import ActionList from "./list-group";
import { setSelectedIds } from "../../../redux/local-states/people-module/action";
import { removeCompanyUserConfirmation } from "../../users/components/confirmation";
import { ACTION_ITEM_TYPE, ACTION_PRIORITY, FILTER_ALL, LOADING_STATES_MODULE, LOG_ACTION } from "../../../utils/constants";
import LoadingState from "../components/loading-state";
import { useIntegrationLoading } from "../../../hooks/Integrations";
import { getCompanySubscriptionRequest } from "../../../redux/subscriptions/action";
import { useGetUserCount } from "../../../hooks/User";
import { getMoreActionItems } from "../../../redux/action-items/action";
import TableLoadingState from "../../../components/loading-states/table";
import helper from "../../../utils/helper";
import { userRequests } from "../../../services/request";
import LoadingButton from "../../../components/loading-button";
import { showFailedAlert, showSuccessAlert } from "../../../utils/alerts";

interface ActionItemsProps {
    actionItems,
    users,
    departments,
    groups,
    actionItemsResponse,
    onRemoveActionItem: (actionItemId: string) => void,
    onRestoreUser: (userId: string) => void,
    getCompanySubscriptionRequest?
}
//: React.FC<ActionItemsProps & PropsFromRedux>
const ActionItems: React.FC<ActionItemsProps & PropsFromRedux> = (props) => {


    const dispatch = useDispatch()
    const history = useHistory();

    const { actionItems, 
            users, 
            departments,
            groups, 
            isLoading, 
            currentUser, 
            actionItemsResponse, 
            loadMore, 
            sortParams, 
            localFilterItems 
        } = props;
    const tmpActionItems = actionItemsSelector;
    const { onRemoveActionItem, sortActionItems, getCompanySubscriptionRequest, getMoreActionItems, getActionItems } = props;
    const { onRestoreUser } = props;
    const { resendActivation } = props;
    const { sort, priorityType, searchKey, selectedIntegrationFilter, selectedModuleFilters, sourceType, isClearFilter } = localFilterItems || {}
    // const [searchKey, setSearchKey] = useState<string>("")
    // const [priorityType, setPriorityType] = useState<string | undefined>(undefined)
    // const [sourceType, setSourceType] = useState<string | undefined>(undefined)
    // const [selectedIntegrationFilter, setSelectedIntegrationFilter] = useState<string | undefined>(undefined)
    // const [selectedModuleFilters, setSelectedModuleFilters] = useState<string[] | undefined>(undefined)
    const [startDate, setStartDate] = useState<Date | null>(null);
    const [endDate, setEndDate] = useState<Date | null>(null);
    // const [sortList, setSortList] = useState(false)
    const [activityItemsList, setActivityItemsList] = useState<any>([]);
    const [isSearching, setIsSearching] = useState<boolean>(false);
    const [isFilteringPriority, setIsFilteringPriority] = useState<boolean>(false)
    const [isFilteringSource, setIsFilteringSource] = useState<boolean>(false)
    const [isFilteringIntegration, setIsFilteringIntegration] = useState<boolean>(false)
    const [isFilteringModule, setIsFilteringModule] = useState<boolean>(false)
    const [isSorting, setIsSorting] = useState<boolean>(false);
    const [searchKeysList, setSearchKeysList] = useState<string[]>([]);
    const [isActionItemReady, setIsActionItemReady] = useState<boolean>(false)
    const [initialLoading, setInitialLoading] = useState<boolean>(true);
    const [showDeleteModal, setShowDeleteModal] = useState(false);
    const [showDismissActionModal, setShowDismissActionModal] = useState(false);
    const [removeDetails, setRemoveDetails] = useState<RemoveCompanyUserParams>({
        userId: "",
        companyId: "",
        name: "",
        actionItemID: ""
    })
    const [buttonLoading, setButtonLoading] = useState(false)
    
    useEffect(() => {
        
        if (_.isEmpty(actionItems)) {
            setInitialLoading(false)
        } else if (!isLoading && activityItemsList.length){
            setInitialLoading(false)
        }
    }, [isLoading, activityItemsList.length])

    useEffect(() => {
        if (isClearFilter) {
            getActionItems({ 
                companyId: currentUser.ActiveCompany, 
                end: "",
                start: "",
                priority: "",
                search_key: searchKey,
                sort: "",
                source: "",
                integration: "",
                module_type: ""
          });
          
        setStartDate(null);
        setEndDate(null);
          dispatch(filterActionItems({
            ...localFilterItems,
            isClearFilter: false,
        }))
        }
    }, [isClearFilter])

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
        // console.log("🚀 ~ handleRemoveUser ~ params:", params)
        
        setRemoveDetails((s) => ({
            ...s,
            userId: params.userId,
            companyId: params.companyId,
            name: params.name,
            actionItemID: params.actionItemID
        }));

        setShowDeleteModal(true);
        
        // removeCompanyUserConfirmation(params).then(result => {
        //     // Handle success if needed
        // }).catch(error => {
           
        // });
    }
    

    

    const displayActionItems = async () => {
        let lists: JSX.Element[] = [];
        let searchKeysItems: string[] = [];
        let actionList = actionItems;
        
        for (let key in actionList) {
            let filteredActions = await actionList[key].filter((res) => {
                return (!priorityType || res.PriorityType === priorityType || !sourceType || !selectedIntegrationFilter)
            });
            if (priorityType) {
                filteredActions = await filteredActions.filter(res => res.PriorityType === priorityType)
            }

            if (sourceType) {
                filteredActions = await filteredActions.filter(res => res?.SourceType?.toLowerCase() === sourceType.toLowerCase())
            }

            if (selectedIntegrationFilter) {
                filteredActions = await filteredActions.filter(res => {
                    return (res?.Log?.LogInfo?.Integration?.ID === selectedIntegrationFilter)
                })
            }

            if (selectedModuleFilters?.length) {
                filteredActions = await filteredActions.filter(res => {
                    return (selectedModuleFilters?.some(m => res?.ActionItemType.includes(m)))
                })
            }

            // if (userCount.active > 1) {
            //     let toBeDismissed = await filteredActions.filter(res => res.ActionItemType === "CREATE_USER")
            //     if (toBeDismissed.length != 0) {
            //         dispatch(deleteActionItem({actionItemId: toBeDismissed[0].ActionItemID, companyId: toBeDismissed[0].CompanyID, autoRemove: true}))
                    
            //     }
            //     filteredActions = await filteredActions.filter(res => res.ActionItemType !== "CREATE_USER")
            // }

            // This array is the indicator if there are available items to search
            // to prevent the hiding of the page filter if there's no search result
            filteredActions?.forEach(item => {
                if (item?.Log?.SearchKey) {
                    searchKeysItems.push(item?.Log?.SearchKey?.toLowerCase())
                }
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
                                    onAddIntegration={(filter) => handleAddIntegration(filter)}
                                    onRemoveActionItem={(id) => onRemoveActionItem(id)}
                                    onImportUsers={() => handleImportIntegrationUsers(item)}
                                    onRestoreUser={() => onRestoreUser(item?.Log?.LogInfo?.Users?.[0]?.Temp?.ID || "")}
                                   
                                />
                            )
                        }
                    }

                })

                if (activities.length) {
                    lists.push(...activities)
                }
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
                                    onAddIntegration={(filter) => handleAddIntegration(filter)}
                                    onRemoveActionItem={(id) => onRemoveActionItem(id)}
                                    onImportUsers={() => handleImportIntegrationUsers(item)}
                                    onRestoreUser={() => onRestoreUser(item?.Log?.LogInfo?.Users?.[0]?.Temp?.ID || item?.Log?.LogInfo?.User?.ID || "")}

                                    onInviteUser={() => handleInviteUser(item?.Log?.LogInfo?.User?.ID || "")}
                                    onRemoveUser={() => {{ handleRemoveUser({ userId: item?.Log?.LogInfo?.User?.ID, companyId: item?.CompanyID, name: item?.Log?.LogInfo?.User?.Name, actionItemID: item.ActionItemID }) }}}
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


        setSearchKeysList(searchKeysItems);
        // return lists;
        setActivityItemsList(lists);
        setIsActionItemReady(true)
        setIsSorting(false)
    }

    


    useEffect(() => {
        sortActionItems({ startDate: 0, endDate: 0 });
        getCompanySubscriptionRequest({ uid: currentUser?.UserID, cid: currentUser?.ActiveCompany })
    }, [])

    useEffect(() => {
        // setTimeout(() => {
            // setIsSorting(false)
            displayActionItems()
        // }, 500)
    }, [searchKey, sort, actionItems, isSearching, priorityType, sourceType, selectedModuleFilters, selectedIntegrationFilter, isSorting])
  
    const handleIntegrationSort = (values) => {
        setIsSorting(true)
        dispatch(filterActionItems({
            ...localFilterItems,
            selectedIntegrationFilter: values,
        }))
        getActionItems({ 
            companyId: currentUser.ActiveCompany, 
            end: sortParams.endDate,
            start: sortParams.startDate,
            priority: priorityType || "",
            search_key: searchKey,
            sort: sort,
            source: sourceType || "",
            integration: values,
            module_type: JSON.stringify(selectedModuleFilters)
        });
        // setSelectedIntegrationFilter(values)
        setIsFilteringIntegration(!!values)
    }

    const handleModuleSort = (values) => {
        setIsSorting(true)
        // setSelectedModuleFilters(values)

        dispatch(filterActionItems({
            ...localFilterItems,
            selectedModuleFilters: values,
        }))
        getActionItems({ 
            companyId: currentUser.ActiveCompany, 
            end: sortParams.endDate,
            start: sortParams.startDate,
            priority: priorityType || "",
            search_key: searchKey,
            sort: sort,
            source: sourceType || "",
            integration: selectedIntegrationFilter || "",
            module_type:  JSON.stringify(values),
        });
        setIsFilteringModule(!!values)
    }

    const handleSourceType = (value) => {
        setIsSorting(true)
        
        dispatch(filterActionItems({
            ...localFilterItems,
            sourceType: value.charAt(0).toUpperCase() + value.slice(1).toLowerCase(),
        }))
        // setSourceType(value)
        getActionItems({ 
            companyId: currentUser.ActiveCompany, 
            end: sortParams.endDate,
            start: sortParams.startDate,
            priority: priorityType || "",
            search_key: searchKey,
            sort: sort,
            source: value.charAt(0).toUpperCase() + value.slice(1).toLowerCase(),
            integration: selectedIntegrationFilter || "",
            module_type: JSON.stringify(selectedModuleFilters)
        });
        setIsFilteringSource(!!value)
    }

    const handlePriorityType = (value) => {
        setIsSorting(true)
        dispatch(filterActionItems({
            ...localFilterItems,
            priorityType: value,
        }))
        // setPriorityType(value)
        getActionItems({ 
            companyId: currentUser.ActiveCompany, 
            end: sortParams.endDate,
            start: sortParams.startDate,
            priority: value,
            search_key: searchKey,
            sort: sort,
            source: sourceType || "",
            integration: selectedIntegrationFilter || "",
            module_type: JSON.stringify(selectedModuleFilters)
        });
        setIsFilteringPriority(!!value)
    }

    const handleDateSort = (sort) => {
        const { startDate, endDate } = sort;
        setStartDate(startDate);
        setEndDate(startDate);
        setIsSorting(true)
        sortActionItems({ startDate: startDate && dayjs(startDate).unix(), endDate: endDate && dayjs(endDate).unix() });
        
        getActionItems({ 
            companyId: currentUser.ActiveCompany, 
            end: endDate && dayjs(endDate).hour(23).minute(59).second(59).unix(),
            start: startDate && dayjs(startDate).unix(),
            priority: priorityType || "",
            search_key: searchKey,
            sort: sort,
            source: sourceType || "",
            integration: selectedIntegrationFilter || "",
            module_type: JSON.stringify(selectedModuleFilters)
        });
        setIsSearching(!!sort)
        setIsFilteringPriority(!!sort)
        setIsFilteringSource(!!sort)
        setIsFilteringModule(!!sort)
        setIsFilteringIntegration(!!sort)
    };

    const handleSortList = (value) => {
        setIsSorting(true);
        setIsSearching(!!value)
        setIsFilteringPriority(!!value)
        setIsFilteringSource(!!value)
        setIsFilteringModule(!!value)
        setIsFilteringIntegration(!!value)
        dispatch(filterActionItems({
            ...localFilterItems,
            sort: value ? "asc" : "desc",
        }))
        getActionItems({ 
            companyId: currentUser.ActiveCompany, 
            end: sortParams.endDate,
            start: sortParams.startDate,
            priority: priorityType || "",
            search_key: searchKey,
            sort: value,
            source: sourceType || "",
            integration: selectedIntegrationFilter || "",
            module_type: JSON.stringify(selectedModuleFilters)
        });
        // setSortList(value);
    }
    const handleSearch = _.debounce((key) => {
        setIsSorting(true)
        dispatch(filterActionItems({
            ...localFilterItems,
            searchKey: key,
        }))
        getActionItems({ 
            companyId: currentUser.ActiveCompany, 
            end: sortParams.endDate,
            start: sortParams.startDate,
            priority: priorityType || "",
            search_key: key,
            sort: sort,
            source: sourceType || "",
            integration: selectedIntegrationFilter || "",
            module_type: JSON.stringify(selectedModuleFilters)
        });
        // setSearchKey(key)
        setIsSearching(!!key)
        setIsFilteringPriority(!!key)
        setIsFilteringSource(!!key)
        setIsFilteringModule(!!key)
        setIsFilteringIntegration(!!key)
    }, 500);

    const integrationsLoading = useIntegrationLoading()
    const fetchMoreActionItems = () => {
        if (!!actionItemsResponse?.lastEvaluatedKey?.PK) {
            getMoreActionItems({
                companyId: currentUser?.ActiveCompany,
                limit: 10,
                last_evaluated_key: {
                    "PK": actionItemsResponse?.lastEvaluatedKey?.PK,
                    "SK": actionItemsResponse?.lastEvaluatedKey?.SK,
                    "GSI_SK": actionItemsResponse?.lastEvaluatedKey?.GSI_SK,
                    "CompanyID": actionItemsResponse?.lastEvaluatedKey?.CompanyID,
                },
                end: sortParams.endDate,
                start: sortParams.startDate,
                priority: priorityType || "",
                search_key: searchKey,
                sort: "",
                source: sourceType || "",
                integration: selectedIntegrationFilter || "",
                module_type: JSON.stringify(selectedModuleFilters),
            });
        }
    }
    const loadMoreActionItems = () => {
        fetchMoreActionItems()
    }
    const handleScroll = (event) => {
      const target = event.target;
      if (!!actionItemsResponse?.lastEvaluatedKey?.PK) {
          if (target.clientHeight + target.scrollTop >= target.scrollHeight - 50) {
            loadMoreActionItems();
          }
      }
    }

    const handleConfirmDeleteUser = (userId, companyId) => {
        setButtonLoading(true)
        const response = userRequests.deleteUser(userId, { company_id: companyId, remove_integration_accounts: true }).then(response => {
            if (response.data.errors.length !== 0 && response.data.errors[0] == "Company user not found") {
                setButtonLoading(false) 
                setShowDeleteModal(false)
                setShowDismissActionModal(true)
            } else {
                setShowDeleteModal(false)
                setButtonLoading(false)
                showSuccessAlert(`${removeDetails.name} has been successfully removed from the company.`)
            }
        }).catch(error => {
            showFailedAlert("An error has occured. Please try again later.")
            return error;
        });
        // console.log("🚀 ~ response ~ response:", response)
        
        
    }

    const dismissActionItem = () => {
        setButtonLoading(true)
        onRemoveActionItem(removeDetails.actionItemID || "")
        setTimeout(() => {
            setButtonLoading(false)
        }, 500);
        setShowDismissActionModal(false)
    }
    
    return (
        <> 
            {
                initialLoading ? (
                    <EmptyStateLoading />
                ) : (
                    !isActionItemReady || integrationsLoading ? (
                        <EmptyStateLoading />
                    ) : (
                        <>
                            {
                                ((isLoading || activityItemsList?.length > 0 || isSearching || searchKeysList.length || isFilteringPriority || isFilteringSource || isFilteringModule || isFilteringIntegration || isFilteringModule
                                    || startDate || endDate || priorityType || sourceType || selectedIntegrationFilter || selectedModuleFilters)) ?
                                    (
                                        <PageFilter
                                            module="dashboard"
                                            onSearch={(key) => handleSearch(key)}
                                            onSort={(sort) => handleDateSort(sort)}
                                            handleSort={(sort) => handleSortList(sort)}
                                            handleSortByPriority={(value) => handlePriorityType(value)}
                                            handleSortBySource={(value) => handleSourceType(value)}
                                            handleSortByIntegration={(value) => handleIntegrationSort(value)}
                                            handleSortByModule={(values) => handleModuleSort(values)}
                                            onFilter={() => { }}
                                            isSearching={isSearching}
                                            isDisabled={!activityItemsList?.length || activityItemsList.length <= 1 }
                                            items={activityItemsList}
                                            setSortLoading={(value) => setIsSorting(value)}

                                            sDate={startDate}
                                            eDate={endDate}
                                            pType={priorityType}
                                            sourceType={sourceType}
                                            selectedIntegrationFilter={selectedIntegrationFilter}
                                            selectedModules={selectedModuleFilters}
                                            
                                        />
                                    ) : <></>
                            }
                            {
                                (isSorting || isLoading) ?
                                    <LoadingState /> :
                                    (
                                        !_.isEmpty(actionItems) && activityItemsList?.length !== 0 ?
                                            (
                                                <Card className="card-custom flex-grow-1 overflow-auto" style={{ zIndex: 1 }}>
                                                    <Card.Body className="p-5 border-bottom-0 d-flex flex-column"  onScroll={handleScroll} >
                                                        <div className="flex-grow-1 overflow-auto custom-scrollbar">
                                                            <Table responsive="sm">
                                                                <thead className="sticky-header">
                                                                    <tr>
                                                                        <th>Summary</th>
                                                                        <th>Priority</th>
                                                                        <th>Source</th>
                                                                        <th className="pl-8">Actions</th>
                                                                    </tr>
                                                                </thead>
                                                                <tbody>
                                                                    {activityItemsList}
                                                                </tbody>
                                                            </Table>
                                                            {
                                                                (loadMore && (
                                                                <Table responsive="sm">
                                                                    <tbody>
                                                                    <tr><td><TableLoadingState module={LOADING_STATES_MODULE.ACTION_ITEMS}  /></td></tr>
                                                                    </tbody>
                                                                </Table>
                                                                ) )
                                                            }
                                                        </div>
                                                    </Card.Body>
                                                </Card>
                                            )
                                        
                                            :
                                            (
                                                
                                                ((users?.length === 0 || departments?.length === 0 || groups?.length === 0)) && !isSearching ?
                                                    <Card className="card-custom h-100 mt-5">
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
                                                        <Card className="card-custom h-100 mt-5">
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
                                                            <Card className="card-custom h-100 mt-5">
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
                )
            }
            <Modal
                show={showDeleteModal}
                scrollable
                onHide={() => setShowDeleteModal(false)}
                centered
                backdrop="static"
                keyboard={false}
            >
                <Modal.Header>
                    <Modal.Title> Remove {removeDetails.name}</Modal.Title>
                </Modal.Header>
                <Modal.Body>
                    <div className="text-left font-weight-boldest">
                        By removing a user from a company, they will also be removed from any associated groups, roles, etc. Are you sure you want to continue?
                    </div>
                </Modal.Body>
                <Modal.Footer>
                    
                    <Button variant="link" onClick={() => setShowDeleteModal(false)} className="text-muted">
                        Close
                    </Button>
                    <LoadingButton loading={buttonLoading} className="sso-button" variant="danger" onClick={() => handleConfirmDeleteUser(removeDetails.userId, removeDetails.companyId)}>
                        Remove
                    </LoadingButton>
                    
                </Modal.Footer>
            </Modal >

            <Modal
                show={showDismissActionModal}
                scrollable
                onHide={() => setShowDismissActionModal(false)}
                centered
                backdrop="static"
                keyboard={false}
            >
                <Modal.Header>
                    <Modal.Title> Dismiss Action Item</Modal.Title>
                </Modal.Header>
                <Modal.Body>
                    <div className="text-left font-weight-boldest">
                        We noticed that you already deleted the user {removeDetails.name}. Do you want to dismiss this action item as well?
                    </div>
                </Modal.Body>
                <Modal.Footer>
                    
                    <Button variant="link" onClick={() => setShowDismissActionModal(false)} className="text-muted">
                        Close
                    </Button>
                    <Button variant="danger" onClick={() => dismissActionItem()} >
                        Dismiss
                    </Button>
                </Modal.Footer>
            </Modal >
        </>
    )
}
const mapDispatchToProps = {
    sortActionItems,
    resendActivation,
    getMoreActionItems,
    getActionItems,
    getCompanySubscriptionRequest,
};

const mapStateToProps = (state: AppState) => {
    return {
        //actionItems: getSortedActionItems(state),
        sortParams: sortSelector(state),
        actionItemsResponse: state.ActionItems.response,
        isLoading: state.ActionItems.isLoading,
        currentUser: state.Auth.user,
        loadMore: state.ActionItems.loadMore,
        localFilterItems : state.ActionItems.localFilterItems,
    }
}

const connector = connect(mapStateToProps, mapDispatchToProps)

type PropsFromRedux = ConnectedProps<typeof connector>

export default connector(ActionItems)
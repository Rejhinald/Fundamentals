import React, { useEffect, useState } from "react";
import { Container, Tab, Tabs } from "react-bootstrap";
import { connect, ConnectedProps, useDispatch } from "react-redux";

import { deleteActionItem, filterActionItems, getActionItems } from "../../redux/action-items/action";
import { listDepartmentRequest } from "../../redux/departments/action";
import { getAllGroups } from "../../redux/groups/action";
import { getCompanyIntegrations, getIntegrations } from "../../redux/integrations/action";
import { AppState } from "../../redux/reducers";
import { getUsers, restoreUsers } from "../../redux/users/action";

import PageBreadcrumbs from "../../components/breadcrumb";
import { } from "../../constants/Constraints";
import { ITEM_STATUS } from "../../utils/constants";
import { getSortedActionItems, sortSelector } from "./selectors";
import "./styles.scss";
import Integrations from "./tabs/integrations";
import { getSubscriptionMembersRequest } from "../../redux/subscriptions/action";
import ActionItems from "./action-items";
import helper from "../../utils/helper";

type PropsFromRedux = ConnectedProps<typeof connector>;

const page = "Dashboard";
const breadcrumbs = [
    {
        title: "Dashboard",
        href: "#",
        active: true,
    }
];
interface ComponentState {
    isRestored: boolean
}

const Dashboard: React.FC<PropsFromRedux> = (props) => {
    const { getUsers, getCompanyIntegrations, listDepartmentRequest, getIntegrations, getAllGroups, getActionItems, deleteActionItem } = props;
    const { restoreUsers } = props;
    const { companyIntegrations, 
            departments, 
            groups, 
            signedInUser, 
            users, 
            actionItems, 
            localFilterItems, 
            actionItemsResponse,
            sortParams 
        } = props;
    const { usersEvents } = props;
    const { sourceType, 
            searchKey, 
            selectedIntegrationFilter, 
            selectedModuleFilters, 
            priorityType, 
            sort,
        } = localFilterItems || {}
    const dispatch = useDispatch()

    //const [loading, setLoading] = useState(true);
    useEffect(() => {
        if (signedInUser?.ActiveCompany) {
            dispatch(getSubscriptionMembersRequest())
            getUsers({ companyId: signedInUser.ActiveCompany })
            getActionItems({ 
                companyId: signedInUser.ActiveCompany, 
                limit: 50,
                end: sortParams.endDate,
                start: sortParams.startDate,
                priority: "",
                search_key: "",
                sort: "",
                source: "",
                integration: "",
                module_type: "[]",
            });
            getCompanyIntegrations({ companyId: signedInUser.ActiveCompany, include: "sub-integrations" });
            getIntegrations({ filter: "categories", include: "sub-integrations" });
            listDepartmentRequest({ includes: "groups" });
            getAllGroups({ company_id: signedInUser.ActiveCompany, status: ITEM_STATUS.ACTIVE, department_id: "", key: "" });
            // refactor set time out to call setLoading to false when all request is finished
            // setTimeout(() => {
            //     setLoading(false);
            // }, 1000);
        }
        return () => {
            //todo: cleanup request
        }
    }, [signedInUser?.ActiveCompany]);

    const [state, setState] = useState<ComponentState>({
        isRestored: false,
    })

    useEffect(() => {
        if (state?.isRestored) {
            if (!usersEvents?.restore?.loading && usersEvents?.restore?.success) {
                getActionItems({ 
                    companyId: signedInUser.ActiveCompany,
                    limit: 8,
                    end: sortParams.endDate,
                    start: sortParams.startDate,
                    priority: priorityType || "",
                    search_key: searchKey,
                    sort: sort,
                    source: sourceType || "",
                    integration: selectedIntegrationFilter,
                    module_type: selectedModuleFilters
                    });
                dispatch(getSubscriptionMembersRequest())
            }
        }
    }, [usersEvents]);

    const [key, setKey] = useState('action-items');


    const handleRemoveActionItem = (actionItemId: string) => {
        deleteActionItem({ actionItemId, companyId: signedInUser?.ActiveCompany });
    }

    const handleRestoreUser = (userId: string) => {
        setState(s => ({ ...s, isRestored: true }));
        dispatch(restoreUsers({ userIds: [userId] }));
    }

    const actionItemProps = {
        actionItems,
        users,
        departments,
        groups,
        onRemoveActionItem: handleRemoveActionItem,
        onRestoreUser: handleRestoreUser,
    }

    return (
        <Container fluid className="main-content dashboard">
            <PageBreadcrumbs items={breadcrumbs} />
            <div>
                <h2 className="page-title font-weight-bolder">{page}</h2>
            </div>
            <ActionItems {...actionItemProps} />
        </Container>
    )
}

const mapDispatchToProps = {
    getCompanyIntegrations,
    listDepartmentRequest,
    getAllGroups,
    getActionItems,
    deleteActionItem,
    getUsers,
    restoreUsers,
    getIntegrations,
};
const mapStateToProps = (state: AppState) => {
    return {
        signedInUser: state.Auth.user,
        companyIntegrations: state.Integrations.companyIntegrations,
        actionItems: getSortedActionItems(state),
        // actionItems: state.ActionItems.actionItems,
        departments: state.Departments.data ?? [], // handle error or empty
        groups: state.Groups.data ?? [], // handle error or empty
        users: state.Users.users,
        sortParams: sortSelector(state),
        actionItemsResponse: state.ActionItems.response,
        usersEvents: state.Users?.events || {},
        localFilterItems : state.ActionItems.localFilterItems,
    };
};
const connector = connect(mapStateToProps, mapDispatchToProps);
export default connector(Dashboard);
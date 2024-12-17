import React, { useState, useEffect } from 'react'
import {
    Container,
    Row,
    Col,
    Accordion
} from 'react-bootstrap'
import { connect, ConnectedProps, useDispatch, useSelector } from 'react-redux';
import { BiCheck, BiMinus, BiX } from 'react-icons/bi'
import { Redirect, useLocation } from 'react-router-dom';
import { IPermission, IRole, ISelect } from '../../interfaces'
import { getUsers } from '../../redux/users/action'
import { AppState } from '../../redux/reducers'
import { deleteRole, getAllRoles, assignRole, sortRole } from '../../redux/roles/action'
import { getAllPermissions } from '../../redux/permissions/action'
import { CATEGORY_PERMISSION, ROLE_MANAGEMENT_PERMISSIONS } from '../../utils/constants'
import { showFailedAlert } from '../../utils/alerts'
import { getSortedPermissions, getSortedRoles, sortSelector } from './selectors';
import PageBreadcrumbs from '../../components/breadcrumb'
import RoleCardComponent from './components/role-table'
import EditModal from './components/modals/edit-modal'
import CreateModal from './components/modals/create-modal'
import AssignModal from './components/modals/assign-modal'
import withReactContent from 'sweetalert2-react-content'
import Swal from 'sweetalert2'
import 'sweetalert2/dist/sweetalert2.css'
import _ from 'lodash'
import './style.scss'
import { ROLE_CONSTANTS } from '../../constants/Role';
import PermissionsPageHeader from './components/page-header';
import { IRoleLocalState } from '../../redux/local-states/roles/types';
import { useAssignRoleResponse, useLoading, useUnAssignRoleResponse } from '../../hooks/Roles';
import { setShowEditRoleModal } from '../../redux/local-states/roles/action';

type PropsFromRedux = ConnectedProps<typeof connector>;

const PermissionsModule: React.FC<PropsFromRedux> = (props) => {
    // redux action
    const {
        getUsers,
        deleteRole,
        assignRole,
        getAllRoles,
        getAllPermissions,
        sortRole
    } = props;
    // redux state
    const {
        users,
        user,
        roles,
        dataLoading,
        loading,
        assignResponse,
        deleteResponse,
        permissions,
        sortParams,
        testSortParams,
    } = props;

    //useState here
    const [selectedIds, SetSelectedIds] = useState<IRole[]>([])
    const [selectedPermissions, SetSelectedPermissions] = useState<string[]>([])
    const [usersData, SetUsersData] = useState<ISelect[]>([])
    const [assignedUsers, SetAssignedUsers] = useState<ISelect[]>([])
    const [assignedUsersCreate, SetAssignedUsersCreate] = useState<ISelect[]>([])
    const [companyPermissions, SetCompanyPermissions] = useState<IPermission[]>([])
    const [companyIntegrationPermissions, SetCompanyIntegrationPermissions] = useState<IPermission[]>([])
    const [companyMemberPermissions, SetCompanyMemberPermissions] = useState<IPermission[]>([])
    const [groupPermissions, SetGroupPermissions] = useState<IPermission[]>([])
    const [groupMemberPermissions, SetGroupMemberPermissions] = useState<IPermission[]>([])
    const [groupIntegrationPermissions, SetGroupIntegrationPermissions] = useState<IPermission[]>([])
    const [departmentPermissions, SetDepartmentPermissions] = useState<IPermission[]>([])
    const [roleManagementPermissions, SetRoleManagementPermissions] = useState<IPermission[]>([])
    const [billingPermissions, SetBillingPermissions] = useState<IPermission[]>([])
    const [selectedCompanyCounter, SetSelectedCompanyCounter] = useState<number>(0)
    const [selectedCompanyIntegrationCounter, SetSelectedCompanyIntegrationCounter] = useState<number>(0)
    const [selectedCompanyMemberCounter, SetSelectedCompanyMemberCounter] = useState<number>(0)
    const [selectedDepartmentCounter, SetSelectedDepartmentCounter] = useState<number>(0)
    const [selectedGroupCounter, SetSelectedGroupCounter] = useState<number>(0)
    const [selectedGroupMemberCounter, SetSelectedGroupMemberCounter] = useState<number>(0)
    const [selectedGroupIntegrationCounter, SetSelectedGroupIntegrationCounter] = useState<number>(0)
    const [selectedRoleManagementCounter, SetSelectedRoleManagementCounter] = useState<number>(0)
    const [selectedBillingCounter, SetSelectedBillingCounter] = useState<number>(0)
    const [editableData, SetEditableData] = useState<any>({})
    const [deleteModal, SetDeleteModal] = useState<boolean>(false)
    const [isChecked, setChecked] = useState<boolean>(true)

    const createModal = useSelector((state: AppState) => state.RoleLocalState.showCreateRoleModal);
    const assignModal = useSelector((state: AppState) => state.RoleLocalState.showAssignRoleModal);
    const editModal = useSelector((state: AppState) => state.RoleLocalState.showEditRoleModal);
    
    //bread crumbs
    const breadcrumbs = [
        {
            title: "Roles",
            href: "#",
            active: true,
        }
    ];


    let location = useLocation();


    const assignRoleResponse = useAssignRoleResponse();
    const unAssignRoleResponse = useUnAssignRoleResponse();
    const roleLoading = useLoading();
    const [isUpdating, setIsUpdating] = useState(false);

    const dispatch = useDispatch();

    useEffect(() => {
        if(!roleLoading && isUpdating) {
            if(assignRoleResponse.success || unAssignRoleResponse.success) {
                dispatch(setShowEditRoleModal(false));
            }
        }
    }, [assignRoleResponse, unAssignRoleResponse, roleLoading]);

    //function to select single role
    const itemOnSelect = (role, categoryCode) => {
        if (selectedIds.some((selected) => selected.RoleID === role.RoleID)) {
            SetSelectedIds(selectedIds.filter((selected) => selected.RoleID !== role.RoleID));
            switch (categoryCode) {
                case CATEGORY_PERMISSION.COMPANY:
                    SetSelectedCompanyCounter(selectedCompanyCounter - 1)
                    break;
                case CATEGORY_PERMISSION.COMPANY_INTEGRATION:
                    SetSelectedCompanyIntegrationCounter(selectedCompanyIntegrationCounter - 1)
                    break;
                case CATEGORY_PERMISSION.COMPANY_MEMBER:
                    SetSelectedCompanyMemberCounter(selectedCompanyMemberCounter - 1)
                case CATEGORY_PERMISSION.DEPARTMENT:
                    SetSelectedDepartmentCounter(selectedDepartmentCounter - 1)
                    break;
                case CATEGORY_PERMISSION.GROUP:
                    SetSelectedGroupCounter(selectedGroupCounter - 1)
                    break;
                case CATEGORY_PERMISSION.GROUP_MEMBER:
                    SetSelectedGroupMemberCounter(selectedGroupMemberCounter - 1)
                    break;
                case CATEGORY_PERMISSION.GROUP_INTEGRATION:
                    SetSelectedGroupIntegrationCounter(selectedGroupIntegrationCounter - 1)
                    break;
                case CATEGORY_PERMISSION.ROLES:
                    SetSelectedRoleManagementCounter(selectedRoleManagementCounter - 1)
                    break;
                case CATEGORY_PERMISSION.BILLING:
                    SetSelectedBillingCounter(selectedBillingCounter - 1)
                    break;
                default:
                    break;
            }

        } else {
            const newIds = [...selectedIds];
            newIds.push(role);
            SetSelectedIds(newIds);
        }
    }

    //function to select all role
    const allOnSelect = (isSelected) => {
        let allIDs: IRole[] = []
        if (!isSelected) {
            roles.forEach((role) => allIDs.push(role))
        } else {
            allIDs = []
        }

        SetSelectedIds(allIDs)
    }

    //function to select all permission in a category
    const allPermissionOnSelect = (isSelected, categoryCode) => {
        let allIDs: string[] = []

        if (!isSelected) {
            switch (categoryCode) {
                case CATEGORY_PERMISSION.COMPANY:
                    companyPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedCompanyCounter(companyPermissions.length)
                    break
                case CATEGORY_PERMISSION.COMPANY_INTEGRATION:
                    companyIntegrationPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedCompanyIntegrationCounter(companyIntegrationPermissions.length)
                    break
                case CATEGORY_PERMISSION.COMPANY_MEMBER:
                    companyMemberPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedCompanyMemberCounter(companyMemberPermissions.length)
                    break
                case CATEGORY_PERMISSION.DEPARTMENT:
                    departmentPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedDepartmentCounter(departmentPermissions.length)
                    break
                case CATEGORY_PERMISSION.GROUP:
                    groupPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedGroupCounter(groupPermissions.length)
                    break
                case CATEGORY_PERMISSION.GROUP_MEMBER:
                    groupMemberPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedGroupMemberCounter(groupMemberPermissions.length)
                    break
                case CATEGORY_PERMISSION.GROUP_INTEGRATION:
                    groupIntegrationPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedGroupIntegrationCounter(groupIntegrationPermissions.length)
                    break
                case CATEGORY_PERMISSION.ROLES:
                    roleManagementPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedRoleManagementCounter(roleManagementPermissions.length)
                    break
                case CATEGORY_PERMISSION.BILLING:
                    billingPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedBillingCounter(billingPermissions.length)
                default:
                    break
            }
            SetSelectedPermissions([...(selectedPermissions || []), ...allIDs])

        } else {
            switch (categoryCode) {
                case CATEGORY_PERMISSION.COMPANY:
                    companyPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedCompanyCounter(0)
                    break
                case CATEGORY_PERMISSION.COMPANY_INTEGRATION:
                    companyIntegrationPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedCompanyIntegrationCounter(0)
                    break
                case CATEGORY_PERMISSION.COMPANY_MEMBER:
                    companyMemberPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedCompanyMemberCounter(0)
                    break
                case CATEGORY_PERMISSION.DEPARTMENT:
                    departmentPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedDepartmentCounter(0)
                    break
                case CATEGORY_PERMISSION.GROUP:
                    groupPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedGroupCounter(0)
                    break
                case CATEGORY_PERMISSION.GROUP_MEMBER:
                    groupMemberPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedGroupMemberCounter(0)
                    break
                case CATEGORY_PERMISSION.GROUP_INTEGRATION:
                    groupIntegrationPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedGroupIntegrationCounter(0)
                case CATEGORY_PERMISSION.ROLES:
                    roleManagementPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedRoleManagementCounter(0)
                    break
                case CATEGORY_PERMISSION.BILLING:
                    billingPermissions.forEach(permission => allIDs.push(permission.PermissionCode))
                    SetSelectedBillingCounter(0)
                default:
                    break
            }

            let result = selectedPermissions?.filter(permission => !allIDs.some(selected => selected === permission))
            SetSelectedPermissions(result)
        }
        if (editModal) setChecked(false)
    }

    //function to count selections on permissions
    const permissionOnSelect = (selection, permissionCode, categoryCode) => {
        if (selection) {
            switch (categoryCode) {
                case CATEGORY_PERMISSION.COMPANY:
                    SetSelectedCompanyCounter(selectedCompanyCounter - 1)
                    break
                case CATEGORY_PERMISSION.COMPANY_INTEGRATION:
                    SetSelectedCompanyIntegrationCounter(selectedCompanyIntegrationCounter - 1)
                    break
                case CATEGORY_PERMISSION.COMPANY_MEMBER:
                    SetSelectedCompanyMemberCounter(selectedCompanyMemberCounter - 1)
                    break
                case CATEGORY_PERMISSION.DEPARTMENT:
                    SetSelectedDepartmentCounter(selectedDepartmentCounter - 1)
                    break
                case CATEGORY_PERMISSION.GROUP:
                    SetSelectedGroupCounter(selectedGroupCounter - 1)
                    break
                case CATEGORY_PERMISSION.GROUP_MEMBER:
                    SetSelectedGroupMemberCounter(selectedGroupMemberCounter - 1)
                    break
                case CATEGORY_PERMISSION.GROUP_INTEGRATION:
                    SetSelectedGroupIntegrationCounter(selectedGroupIntegrationCounter - 1)
                    break
                case CATEGORY_PERMISSION.ROLES:
                    SetSelectedRoleManagementCounter(selectedRoleManagementCounter - 1)
                    break
                case CATEGORY_PERMISSION.BILLING:
                    SetSelectedBillingCounter(selectedBillingCounter - 1)
                    break
                default:
                    break
            }
            SetSelectedPermissions(selectedPermissions?.filter(permission => permissionCode !== permission))
        } else {

            switch (categoryCode) {
                case CATEGORY_PERMISSION.COMPANY:
                    SetSelectedCompanyCounter(selectedCompanyCounter + 1)
                    break
                case CATEGORY_PERMISSION.COMPANY_INTEGRATION:
                    SetSelectedCompanyIntegrationCounter(selectedCompanyIntegrationCounter + 1)
                    break
                case CATEGORY_PERMISSION.COMPANY_MEMBER:
                    SetSelectedCompanyMemberCounter(selectedCompanyMemberCounter + 1)
                    break
                case CATEGORY_PERMISSION.DEPARTMENT:
                    SetSelectedDepartmentCounter(selectedDepartmentCounter + 1)
                    break
                case CATEGORY_PERMISSION.GROUP:
                    SetSelectedGroupCounter(selectedGroupCounter + 1)
                    break
                case CATEGORY_PERMISSION.GROUP_MEMBER:
                    SetSelectedGroupMemberCounter(selectedGroupMemberCounter + 1)
                    break
                case CATEGORY_PERMISSION.GROUP_INTEGRATION:
                    SetSelectedGroupIntegrationCounter(selectedGroupIntegrationCounter + 1)
                    break
                case CATEGORY_PERMISSION.ROLES:
                    SetSelectedRoleManagementCounter(selectedRoleManagementCounter + 1)
                    break
                case CATEGORY_PERMISSION.BILLING:
                    SetSelectedBillingCounter(selectedBillingCounter + 1)
                    break
                default:
                    break
            }
            SetSelectedPermissions([...(selectedPermissions ?? []), permissionCode])
        }
        if (editModal) setChecked(false)
    }

    //function to fetch information on click
    const EditOnClickSelect = async (role: IRole) => {
        const existingDepartment = role.RolePermissions ? role.RolePermissions.filter(permission => departmentPermissions.some(department => permission === department.PermissionCode)) : []
        const existingGroups = role.RolePermissions ? role.RolePermissions.filter(permission => groupPermissions.some(group => permission === group.PermissionCode)) : []
        const existingGroupMembers = role.RolePermissions ? role.RolePermissions.filter(permission => groupMemberPermissions.some(groupMember => permission === groupMember.PermissionCode)) : []
        const existingGroupIntegrations = role.RolePermissions ? role.RolePermissions.filter(permission => groupIntegrationPermissions.some(groupIntegration => permission === groupIntegration.PermissionCode)) : []
        const existingCompany = role.RolePermissions ? role.RolePermissions.filter(permission => companyPermissions.some(company => permission === company.PermissionCode)) : []
        const existingCompanyIntegrations = role.RolePermissions ? role.RolePermissions.filter(permission => companyIntegrationPermissions.some(companyIntegration => permission === companyIntegration.PermissionCode)) : []
        const existingCompanyMembers = role.RolePermissions ? role.RolePermissions.filter(permission => companyMemberPermissions.some(companyMember => permission === companyMember.PermissionCode)) : []
        const existingRoles = role.RolePermissions ? role.RolePermissions.filter(permission => roleManagementPermissions.some(roleAction => permission === roleAction.PermissionCode)) : []
        const existingBilling = role.RolePermissions ? role.RolePermissions.filter(permission => billingPermissions.some(billing => permission === billing.PermissionCode)) : []

        // const formattedAssignedUser = role.Users ? role.Users.filter(u => u.UserID !== user.UserID ).map(u => {
        //     return {
        //         label: u.FirstName + " " + u.LastName,
        //         value: u.UserID
        //     }
        // }) : []


        const tmpCurrentUser = role?.Users?.filter(u => u.UserID == user.UserID) || [];
        const tmpAssignedUsers = role?.Users?.filter(u => u.UserID !== user.UserID) || [];
        const formattedAssignedUser = [...tmpCurrentUser, ...tmpAssignedUsers]?.map(u => ({
            label: u.FirstName + " " + u.LastName,
            value: u.UserID,
            isFixed: (u.UserType === "COMPANY_OWNER") || 
                (ROLE_CONSTANTS.COMPANY_ADMIN_ROLE === role.RoleName && u.UserID === user.UserID),
            userType: u.UserType
        })) || [];
    
        await SetAssignedUsers(formattedAssignedUser)

        await SetSelectedDepartmentCounter(existingDepartment.length)
        await SetSelectedGroupCounter(existingGroups.length)
        await SetSelectedGroupMemberCounter(existingGroupMembers.length)
        await SetSelectedGroupIntegrationCounter(existingGroupIntegrations.length)
        await SetSelectedCompanyCounter(existingCompany.length)
        await SetSelectedCompanyIntegrationCounter(existingCompanyIntegrations.length)
        await SetSelectedCompanyMemberCounter(existingCompanyMembers.length)
        await SetSelectedRoleManagementCounter(existingRoles.length)
        await SetSelectedBillingCounter(existingBilling.length)
        await SetEditableData(role)

        await SetSelectedPermissions(role.RolePermissions)
        //await SetEditModal(true)

    }

    //check to return if category is check
    const isAllPermissionCategorySelected = (category) => {
        switch (category) {
            case CATEGORY_PERMISSION.COMPANY:
                return selectedCompanyCounter === companyPermissions.length ? true : false
            case CATEGORY_PERMISSION.COMPANY_INTEGRATION:
                return selectedCompanyIntegrationCounter === companyIntegrationPermissions.length ? true : false
            case CATEGORY_PERMISSION.COMPANY_MEMBER:
                return selectedCompanyMemberCounter === companyMemberPermissions.length ? true : false
            case CATEGORY_PERMISSION.DEPARTMENT:
                return selectedDepartmentCounter === departmentPermissions.length ? true : false
            case CATEGORY_PERMISSION.GROUP:
                return selectedGroupCounter === groupPermissions.length ? true : false
            case CATEGORY_PERMISSION.GROUP_MEMBER:
                return selectedGroupMemberCounter === groupMemberPermissions.length ? true : false
            case CATEGORY_PERMISSION.GROUP_INTEGRATION:
                return selectedGroupIntegrationCounter === groupIntegrationPermissions.length ? true : false
            case CATEGORY_PERMISSION.ROLES:
                return selectedRoleManagementCounter === roleManagementPermissions.length ? true : false
            case CATEGORY_PERMISSION.BILLING:
                return selectedBillingCounter === billingPermissions.length ? true : false
            default:
                break;
        }

        return false
    }

    //check to return is selected permissions
    const isPermissionSelected = (rolePermission) => {
        return selectedPermissions?.some(permission => permission === rolePermission)
    }
    
    //calculate permission for icon ui
    const CalculatePermissions = (category, rolePermissions) => {
        let count: number
        switch (category) {
            case CATEGORY_PERMISSION.COMPANY:
                count = (companyPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === companyPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.COMPANY_INTEGRATION:
                count = (companyIntegrationPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === companyIntegrationPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.COMPANY_MEMBER:
                count = (companyMemberPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === companyMemberPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.DEPARTMENT:
                count = (departmentPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === departmentPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.GROUP:
                count = (groupPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === groupPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.GROUP_MEMBER:
                count = (groupMemberPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === groupMemberPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.GROUP_INTEGRATION:
                count = (groupIntegrationPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === groupIntegrationPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.ROLES:
                count = (roleManagementPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === roleManagementPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            case CATEGORY_PERMISSION.BILLING:
                count = (billingPermissions.filter((permission) => rolePermissions.some(role => role === permission.PermissionCode))).length
                if (count === billingPermissions.length) {
                    return <BiCheck color="green" aria-label='green Check Icon'/>
                } else if (count !== 0) {
                    return <BiMinus color="gray" aria-label='Gray Minus Icon'/>
                } else {
                    return <BiX color="red" aria-label='Red X Close Icon'/>
                }
                break;
            default:
                break;
        }
    }

    //search function
    const handleSearchInput = _.debounce((key) => {
        getAllRoles({ company_id: user.ActiveCompany, key: key })
    }, 500);

    const handleAssignRoleOnClick = (values) => {
        const data = new FormData()
        const { selectedRoles, selectedUsers} = values || {};
        
        selectedRoles.forEach((role) => {
            data.append("role_id[]", role.RoleID)
        })

        selectedUsers.map((user) => {
            data.append("user_id[]", user.value)
        })

        assignRole({ form: data, company_id: user.ActiveCompany })
        // SetAssignModal(false)
    }

    const handleSortBy = (sortKey) => {
        const order = sortParams?.sortOrder === "asc" || sortParams === undefined ? "desc" : "asc";
        sortRole({ sortKey, sortOrder: order })
    }

    //all side effects here
    useEffect(() => {

        if (assignResponse.status?.code) {
            switch (assignResponse.status?.code) {
                case "500":
                    showFailedAlert('Server error. Please contact support')
                    break
                default:
                    //SetAssignModal(false)
                    break
            }
        }
    }, [assignResponse])

    useEffect(() => {

        if (deleteResponse.status?.code) {
            switch (deleteResponse.status?.code) {
                case "500":
                    showFailedAlert('Server error. Please contact support')
                    SetDeleteModal(false)
                    break
                case "200":
                    SetDeleteModal(false)
                    break
                default:
                    SetDeleteModal(false)
                    break
            }
        }

    }, [deleteResponse])

    useEffect(() => {
        if (deleteModal) {
            const MySwal = withReactContent(Swal)
            MySwal.fire({
                icon: "warning",
                title: `Remove Role`,
                text: `Are you sure you want to remove the selected role?`,
                showCancelButton: true,
                heightAuto: false,
                confirmButtonText: "Remove",
                customClass: {
                    confirmButton: "btn btn-danger order-2",
                    cancelButton: "btn btn-link text-muted"
                },
                buttonsStyling: false
            }).then(async (result) => {
                if (result.isConfirmed) {
                    SetSelectedIds([])
                    const formData = new FormData()
                    selectedIds.map((role) => {
                        formData.append("role_id[]", role.RoleID)
                    })
                    formData.append("company_id", user.ActiveCompany)
                    await deleteRole({ form: formData, company_id: user.ActiveCompany })
                } else {
                    SetDeleteModal(false)
                }
            })
        }
    }, [deleteModal])

    useEffect(() => {
        if (createModal) {
            SetSelectedPermissions([])
            SetSelectedCompanyCounter(0)
            SetSelectedCompanyIntegrationCounter(0)
            SetSelectedCompanyMemberCounter(0)
            SetSelectedGroupCounter(0)
            SetSelectedGroupMemberCounter(0)
            SetSelectedGroupIntegrationCounter(0)
            SetSelectedDepartmentCounter(0)
            SetSelectedRoleManagementCounter(0)
            SetSelectedBillingCounter(0)
        }
    }, [createModal])

    useEffect(() => {
        getAllPermissions()
        getAllRoles({ company_id: user.ActiveCompany })
    }, [user.ActiveCompany])

    useEffect(() => {
        if (permissions.length) {
            SetCompanyPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.COMPANY))
            SetCompanyIntegrationPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.COMPANY_INTEGRATION))
            SetCompanyMemberPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.COMPANY_MEMBER))
            SetGroupPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.GROUP))
            SetGroupMemberPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.GROUP_MEMBER))
            SetGroupIntegrationPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.GROUP_INTEGRATION))
            SetDepartmentPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.DEPARTMENT))
            SetRoleManagementPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.ROLES))
            SetBillingPermissions(permissions.filter((value) => value.PermissionCategoryCode === CATEGORY_PERMISSION.BILLING))
        }
    }, [permissions])

    useEffect(() => {
        if (users.length) {
            console.log('All Users Data:', users);
            const formattedUser = users?.map(u => ({
                label: u.FirstName + " " + u.LastName,
                value: u.UserID,
                isFixed: u.UserType === "COMPANY_OWNER" ||  // Changed to use UserType check
                        u.UserID === user.UserID,
                UserType: u.UserType  // Note: Changed to uppercase U in UserType
            })) || [];
            SetUsersData(formattedUser)
        }
    }, [users])

    useEffect(() => {
        getUsers({ companyId: user.ActiveCompany, include: "groups,roles" })
    }, [user.ActiveCompany])


    // if (!user?.IsAdmin) {
    //     return (
    //         <Redirect to={{
    //             pathname: "/dashboard",
    //             state: { from: location },
    //         }} />
    //     )
    // }

    if (user?.Permissions !== undefined) {
        if (!(ROLE_MANAGEMENT_PERMISSIONS.some(role => user?.Permissions.includes(role)))) {
            return <Redirect to={{
                pathname: "/dashboard",
                state: { from: location },
            }} />
        }
    }
    
    return (
        <>
            <Container fluid className="main-content permissions">
                <Row>
                    <Col>
                        <PageBreadcrumbs items={breadcrumbs} />
                    </Col>
                </Row>
                <Row>
                    <Col className="w-100">
                        <PermissionsPageHeader
                            onDeleteRole={SetDeleteModal}
                            selectedIds={selectedIds}
                            userPermissions={user?.Permissions}
                        />
                    </Col>
                </Row>
                <Row className="pt-5 h-100">
                    <Col>
                        <RoleCardComponent
                            roles={roles}
                            isLoading={dataLoading}
                            selectedIds={selectedIds}
                            allOnSelect={allOnSelect}
                            itemOnSelect={itemOnSelect}
                            CalculatePermissions={CalculatePermissions}
                            EditOnClickSelect={EditOnClickSelect}
                            SetEditableData={SetEditableData}
                            handleSearchInput={handleSearchInput}
                            userPermissions={user?.Permissions}
                            handleSort={handleSortBy}
                        />
                    </Col>
                </Row>
            </Container>
            {
                editModal && (
                <EditModal
                    show={editModal}
                    isLoading={loading}
                    role={editableData}
                    permissions={permissions}
                    usersData={usersData}
                    selectedPermissions={selectedPermissions}
                    assignedUsers={assignedUsers}
                    SetAssignedUsers={SetAssignedUsers}
                    permissionOnSelect={permissionOnSelect}
                    isPermissionSelected={isPermissionSelected}
                    allPermissionOnSelect={allPermissionOnSelect}
                    isAllPermissionCategorySelected={isAllPermissionCategorySelected}
                    isChecked={isChecked}
                    disabledButton={setChecked}
                    userPermissions={user?.Permissions}
                    currentUser={user?.UserID}
                    setIsUpdating={setIsUpdating}
                    userType={user?.UserType}
                    users={users}  
                />
                )
            }
            {
                createModal && (
                    <CreateModal
                        show={createModal}
                        isLoading={loading}
                        permissions={permissions}
                        usersData={usersData}
                        assignedUsers={assignedUsersCreate}
                        currentUser={user}
                        selectedPermissions={selectedPermissions}
                        SetAssignedUsers={SetAssignedUsersCreate}
                        permissionOnSelect={permissionOnSelect}
                        allPermissionOnSelect={allPermissionOnSelect}
                        isPermissionSelected={isPermissionSelected}
                        isAllPermissionCategorySelected={isAllPermissionCategorySelected}
                    // handleUpdateOnClick={handleUpdateOnClick}
                    />
                )
            }
            {
                assignModal && (
                    <AssignModal
                        show={assignModal}
                        isLoading={loading}
                        selectedRoles={selectedIds}
                        users={users}
                        roles={roles}
                        handleAssignRoleOnClick={handleAssignRoleOnClick}
                    />
                )
            }
        </>
    )
}


const mapDispatchToProps = {
    getUsers,
    deleteRole,
    assignRole,
    getAllRoles,
    getAllPermissions,
    sortRole,
};

const mapStateToProps = (state: AppState) => {
    return {
        users: state.Users.users ?? [],
        user: state.Auth.user,
        roles: getSortedRoles(state),
        dataLoading: state.Roles.dataLoading ?? false,
        loading: state.Roles.loading ?? false,
        assignResponse: state.Roles.assignResponse ?? false,
        deleteResponse: state.Roles.deleteResponse ?? false,
        permissions: getSortedPermissions(state),
        sortParams: sortSelector(state),
        testSortParams: state.Roles.sortParams
    }
}

const connector = connect(mapStateToProps, mapDispatchToProps);

export default connector(PermissionsModule);
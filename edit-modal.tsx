import React, {
    useEffect,
    useState,
} from 'react'

import {
    Modal,
    Button,
    Form,
    Row,
    Col,
    Card,
    OverlayTrigger,
    Tooltip
} from 'react-bootstrap'

import {
    IRole,
    ISelect,
    IPermission
} from '../../../../interfaces'

import {
    useDispatch,
    useSelector
} from 'react-redux'

import {
    assignRole,
    unassignRole,
    assignPermission
} from '../../../../redux/roles/action'

import {
    CATEGORY_PERMISSIONS_ARRAY
} from '../../../../utils/constants'

import { AppState } from '../../../../redux/reducers'

import { showFailedAlert } from '../../../../utils/alerts'

import { OptionsType, StylesConfig } from 'react-select'

import { useFormik } from 'formik'

import LoadingButton from '../../../../components/loading-button'
import Select from 'react-select'
import * as yup from 'yup'
import './style.scss'
import { IoCaretDown, IoCaretUp } from 'react-icons/io5'
import SelectionCheckbox from '../../../../components/inputs/checkbox'
import { PERMISSIONS } from '../../../../constants/Permissions';
import { VALIDATION_MESSAGES } from '../../../../constants/forms/ValidationMessages';
import _ from "lodash";
import { ROLE_CONSTANTS } from '../../../../constants/Role'
import InformationTooltip from '../../../../components/tooltips/information'
import { Dispatch } from 'redux'
import { setShowEditRoleModal } from '../../../../redux/local-states/roles/action'

interface IEditModal {
    show: boolean
    isLoading: boolean
    role: IRole
    permissions: IPermission[]
    usersData: ISelect[]
    assignedUsers: ISelect[]
    selectedPermissions: string[]
    SetAssignedUsers: (selected: any) => void
    permissionOnSelect: (selection: boolean, roleID: string, categoryCode: string) => void
    allPermissionOnSelect: (isSelected: boolean, categoryCode: string) => void
    isPermissionSelected: (rolePermission: string) => boolean
    isAllPermissionCategorySelected: (category: string) => boolean
    isChecked: boolean
    disabledButton: (value: boolean) => void,
    userPermissions: string[],
    currentUser: string,
    setIsUpdating: (val: boolean) => void,
}

const EditModal = ({
    show,
    role,
    isLoading,
    permissions,
    usersData,
    assignedUsers,
    selectedPermissions,
    SetAssignedUsers,
    isPermissionSelected,
    isAllPermissionCategorySelected,
    permissionOnSelect,
    allPermissionOnSelect,
    isChecked,
    disabledButton,
    userPermissions,
    currentUser,
    setIsUpdating,
}: IEditModal) => {
    const userPemissionsDefault: number = selectedPermissions?.length;
    const dispatch: Dispatch = useDispatch()

    const user = useSelector((state: AppState) => state.Auth.user)
    const updatedRole = useSelector((state: AppState) => state.Roles.response ?? [])
    const [activePanel, setActivePanel] = useState<number | any>(undefined)
    const SYSTEM = "SYSTEM";

    const togglePanel = (index, toggle = true) => {
        if (toggle) setActivePanel(activePanel === index ? undefined : index)
    }

    const handleOnSelectPermissionSet = (index, isSelected, categoryCode) => {
        allPermissionOnSelect(isSelected, categoryCode);
        setActivePanel(index)
    }

    const checkSelectedPermissions = (categoryCode): boolean => {
        let counter: number = 0;
        const filteredPermissions = permissions.filter(permission => permission.PermissionCategoryCode === categoryCode ? permission : null);
        filteredPermissions.forEach((permission => {
            if (isPermissionSelected(permission.PermissionCode)) {
                counter++;
            }
        }));

        if (counter > 0) return true;
        return false;
    }

    const sortedCategoryPermissions = _.sortBy(CATEGORY_PERMISSIONS_ARRAY, [(category) => category.NAME])

    const sortedUsers = _.sortBy(usersData, [(user) => user.label])

    const Formik = useFormik({
        initialValues: {
            roleID: role.RoleID ?? '',
            roleName: role.RoleName ?? '',
            selectedUsers: assignedUsers ?? []
        },
        validationSchema: yup.object({
            roleName: yup.string()
                .required(VALIDATION_MESSAGES.required("Name"))
                .max(50, 'Must be 50 characters or less')
                .matches('^[a-zA-Z0-9 ]*$', 'No special characters allowed')
        }),
        validateOnBlur: true,
        enableReinitialize: true,
        onSubmit: async (values) => {

            setIsUpdating(true);

            const currentllyAssigend = assignedUsers as { value: string, label: string, isFixed: boolean, userRole?: string[] }[]
            const notFixed = currentllyAssigend?.filter(user => !user?.isFixed)

            const toRemoveFromRole = notFixed?.filter(cur => !values.selectedUsers?.map(a => a.value).includes(cur.value)) || []
            const toAddFromRole = values.selectedUsers?.filter(b => !currentllyAssigend.map(a => a.value).includes(b.value)) || []


            const previousUsers = role.Users ? role.Users : []
            const currentUser = values.selectedUsers ? values.selectedUsers : []
            const data = new FormData()
            data.append("role_id", values.roleID)
            data.append("role_name", values.roleName)
            data.append("company_id", user.ActiveCompany)

            selectedPermissions?.forEach((value) => {
                data.append("role_permission[]", value)
            })

            await dispatch(assignPermission({ company_id: user.ActiveCompany, form: data, skipAlert: previousUsers.length !== currentUser.length }))

            if (toRemoveFromRole.length) {
                

                // const filteredUsers = previousUsers.filter(prev => !currentUser.some(user => prev.UserID === user.value))
                const roleData = new FormData()
                roleData.append("role_id[]", values.roleID)
                toRemoveFromRole.forEach((user) => {
                    roleData.append("user_id[]", user.value)
                })

                await dispatch(unassignRole({ form: roleData, company_id: user.ActiveCompany }))

            }

            if (toAddFromRole.length) {
                
                const roleData = new FormData()
                roleData.append("role_id[]", values.roleID)
                toAddFromRole.forEach((user) => {
                    roleData.append("user_id[]", user.value)
                })

                await dispatch(assignRole({ form: roleData, company_id: user.ActiveCompany }))
            }

            
            // const previousUsers = role.Users ? role.Users : []
            // const currentUser = values.selectedUsers ? values.selectedUsers : []
            // const data = new FormData()
            // data.append("role_id", values.roleID)
            // data.append("role_name", values.roleName)
            // data.append("company_id", user.ActiveCompany)

            // selectedPermissions?.forEach((value) => {
            //     data.append("role_permission[]", value)
            // })

            // await dispatch(assignPermission({ company_id: user.ActiveCompany, form: data, skipAlert: previousUsers.length != currentUser.length }))

            // if (previousUsers.length > currentUser.length) {
            //     const filteredUsers = previousUsers.filter(prev => !currentUser.some(user => prev.UserID === user.value))
            //     const roleData = new FormData()

            //     roleData.append("role_id[]", values.roleID)

            //     filteredUsers.map((user) => {
            //         roleData.append("user_id[]", user.UserID)
            //     })

            //     await dispatch(unassignRole({ form: roleData, company_id: user.ActiveCompany }))

            // }

            // if (previousUsers.length < currentUser.length) {
            //     const roleData = new FormData()

            //     roleData.append("role_id[]", values.roleID)

            //     currentUser.map((user) => {
            //         roleData.append("user_id[]", user.value)
            //     })

            //     await dispatch(assignRole({ form: roleData, company_id: user.ActiveCompany }))
            // }
        },
    })
    // useEffect(() => {
    //     if (updatedRole.status?.code) {
    //         switch (updatedRole.status?.code) {
    //             case "422":
    //                 Formik.setFieldError('roleName', updatedRole.errors)
    //                 break
    //             case "500":
    //                 showFailedAlert('Server error. Please contact support')
    //                 break
    //             default:
    //                 //handleClose()
    //                 break
    //         }
    //     }
    // }, [updatedRole])

    const styles = {
        multiValue: (base, state) => {
            return role.RoleName === ROLE_CONSTANTS.COMPANY_ADMIN_ROLE && state.data.isFixed ? { ...base, backgroundColor: 'gray' } : base;
        },
        multiValueLabel: (base, state) => {
            return role.RoleName === ROLE_CONSTANTS.COMPANY_ADMIN_ROLE && state.data.isFixed
                ? { ...base, fontWeight: 'bold', color: 'white', backgroundColor: 'var(--primary)', paddingRight: 6 }
                : base;
        },
        multiValueRemove: (base, state) => {
            return role.RoleName === ROLE_CONSTANTS.COMPANY_ADMIN_ROLE && state.data.isFixed ? { ...base, display: 'none' } : base;
        },
    };

    const onClose = () => {
        Formik.resetForm();
        setActivePanel(undefined);
        disabledButton(true);
        dispatch(setShowEditRoleModal(false));
    }

    return (
        <Modal
            show={show}
            centered
            // className="custom-permission-modal"
            scrollable
        >
            <Modal.Header closeButton>
                <Modal.Title>Edit {role.RoleName}</Modal.Title>
            </Modal.Header>

            <Modal.Body className="custom-scrollbar">
                <Row>
                    <Col>
                        <Form.Group className="mb-3">
                            <Form.Control type="hidden" id="roleID" name="roleID" value={Formik.values.roleID} />
                            <Form.Label className="font-weight-bolder">Name <span className="required-field">*</span></Form.Label>
                            <Form.Control
                                type="input"
                                id="roleName"
                                name="roleName"
                                placeholder="Enter Name"
                                value={Formik.values.roleName}
                                onChange={Formik.handleChange}
                                onBlur={Formik.handleBlur}
                                isInvalid={Formik.touched.roleName && Formik.errors.roleName ? true : false}
                                disabled={role.CompanyID === SYSTEM || isLoading || !(userPermissions?.includes(PERMISSIONS.EDIT_ROLE) && userPermissions?.includes(PERMISSIONS.UNASSIGN_ROLE))}
                                aria-label="Edit Role Name"
                            />
                            <Form.Control.Feedback type='invalid'>
                                {Formik.errors.roleName}
                            </Form.Control.Feedback>
                        </Form.Group>

                    </Col>
                </Row>
                <Row>
                    <Col>
                        <Form.Group className="mb-3">
                            <Form.Control type="hidden" id="users" name="users" value={Formik.values.roleID} />
                            <Form.Label className="font-weight-bolder">Users
                                { 
                                    Formik.values.roleName === ROLE_CONSTANTS.COMPANY_ADMIN_ROLE ? 
                                        userPermissions.length < 26 ?
                                        <InformationTooltip>To be able to remove users as Company Admin you must have that role.</InformationTooltip> :
                                        Formik.values.selectedUsers.length === 1 ?
                                        <InformationTooltip>You are unable to remove yourself with the Company Admin role because your company requires at least one.</InformationTooltip> 
                                        : <></>
                                    : <></>
                                }
                            </Form.Label>
                            
                            <Select
                                id="selectedUsers"
                                name="selectedUsers"
                                value={Formik.values.selectedUsers}
                                options={sortedUsers}
                                isDisabled={!(userPermissions?.includes(PERMISSIONS.UNASSIGN_ROLE))}
                                isMulti
                                onChange={(option: any, actionMeta: any) => {
                                    switch (actionMeta.action) {
                                        case 'remove-value':
                                        case 'pop-value':
                                            if (actionMeta.removedValue.isFixed && role.RoleName === ROLE_CONSTANTS.COMPANY_ADMIN_ROLE) {
                                                return;
                                            }
                                            break;
                                        case 'clear':
                                            option = usersData.filter((v: any) => {
                                                if (role.RoleName === ROLE_CONSTANTS.COMPANY_ADMIN_ROLE) {
                                                    return v?.userRole === undefined
                                                } else {
                                                    return ""
                                                }
                                            });
                                            break;
                                    }
                                    Formik.setFieldValue('selectedUsers', option)
                                }}
                                onBlur={Formik.handleBlur}
                                className="react-select"
                                isClearable={role.RoleName === ROLE_CONSTANTS.COMPANY_ADMIN_ROLE ? false : usersData?.some((v: any) => !v.isFixed)}

                                styles={styles}
                                placeholder="Select Users"
                            // isInvalid={Formik.touched.roleName && Formik.errors.roleName ? true : false}
                                aria-label="Select users"
                            />
                            <Form.Control.Feedback type='invalid'>
                                {Formik.errors.roleName}
                            </Form.Control.Feedback>
                        </Form.Group>

                    </Col>
                </Row>
                <Row>
                    <Col>
                        <Form.Label className="font-weight-bolder">Permissions List</Form.Label>
                        <div className={`permissions-list border rounded py-1 ${!(userPermissions?.includes(PERMISSIONS.EDIT_ROLE)) ? "prevent-editing-roles" : ""}`}>
                            {sortedCategoryPermissions.map((category, index) => (
                                <Card key={index} className="border-0">
                                    <Card.Header className="bg-gray px-5 py-2" onClick={() => togglePanel(index)}>
                                        <div className="d-flex justify-content-between align-items-center">
                                            <div className="d-flex">
                                                <SelectionCheckbox
                                                    onChange={() => handleOnSelectPermissionSet(index, isAllPermissionCategorySelected(category.CODE), category.CODE)}
                                                    isSelected={isAllPermissionCategorySelected(category.CODE)}
                                                    disabled={(role.CompanyID === SYSTEM || isLoading || !(userPermissions?.includes(PERMISSIONS.EDIT_ROLE)))}
                                                />
                                                <span className="pl-2 font-weight-bolder">{category.NAME}</span>
                                            </div>
                                            {
                                                (checkSelectedPermissions(category.CODE) || activePanel === index) ? <IoCaretUp className="accordion-btn accordion-carret" /> : <IoCaretDown className="accordion-btn accordion-carret" />
                                            }
                                        </div>
                                    </Card.Header>
                                    <div className={`collapse ${(checkSelectedPermissions(category.CODE) || activePanel === index) ? "show" : ""}`}>

                                        <Card.Body className="pt-1 pb-2 border-bottom-0 d-flex flex-column">
                                            {permissions.filter(permission => permission.PermissionCategoryCode === category.CODE ? permission : null)
                                                .map((permission, index) => (
                                                    <div key={index} id={permission.PK} className="d-flex align-items-center py-1">
                                                        <SelectionCheckbox
                                                            onChange={() => permissionOnSelect(isPermissionSelected(permission.PermissionCode), permission.PermissionCode, permission.PermissionCategoryCode)}
                                                            isSelected={isPermissionSelected(permission.PermissionCode) ?? false}
                                                            disabled={role.CompanyID === SYSTEM || isLoading || !(userPermissions?.includes(PERMISSIONS.EDIT_ROLE))}
                                                        />
                                                        <span className="pl-2">{permission.PermissionName}</span>
                                                    </div>
                                                ))}
                                        </Card.Body>
                                    </div>
                                </Card>
                            ))}
                        </div>
                    </Col>
                </Row>
            </Modal.Body>

            <Modal.Footer className="border-top-0">
                <Button variant="link" className='text-muted' onClick={onClose} disabled={isLoading}>
                    Cancel
                </Button>
                <LoadingButton loading={isLoading} variant="primary" disabled={!(Formik.isValid && Formik.dirty) && isChecked} onClick={() => {
                    Formik.handleSubmit();
                    disabledButton(true);
                }}>
                    Save
                </LoadingButton>

            </Modal.Footer>
        </Modal>
    )
}

export default EditModal;
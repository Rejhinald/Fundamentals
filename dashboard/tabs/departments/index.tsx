import React from "react";
import { Card, Table, Button } from "react-bootstrap";
import { Link } from "react-router-dom";
import GroupAvatar from "../../../../components/group-avatar";
import Avatar from "../../../../components/avatar";
import { BsPlus } from "react-icons/bs";

interface DepartmentsProps {
    departments,
}

const Departments: React.FC<DepartmentsProps> = (props) => {
    const { departments } = props;
    return (
        <Card className="card-custom flex-grow-1 h-100 overflow-auto custom-scrollbar">
            <Card.Body className="p-5 border-bottom-0">
                {
                    departments?.length ?
                        <div className="table-responsive">
                            <Table className="user-table" responsive>
                                <thead>
                                    <tr>
                                        <th>Department</th>
                                        <th>Groups</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {
                                        departments.map((department, idx) => (
                                            <tr key={idx}>
                                                <td>
                                                    <Link
                                                        to={`/departments/${department.DepartmentID}`}
                                                    >
                                                        <Avatar
                                                            name={department.DepartmentName}
                                                        />
                                                    </Link>
                                                        <span className="ml-3">{department.DepartmentName}</span>
                                                </td>
                                                    <td>
                                                        {
                                                            ("Groups" in department) ?
                                                                department.Groups.map((group, idx) => (
                                                                    <Link
                                                                        key={idx}
                                                                        to={`/groups/${group.GroupID}`}
                                                                    >
                                                                        <Avatar
                                                                            name={group?.GroupName}
                                                                            className="avatar--small  "
                                                                            style={{
                                                                                marginLeft: idx != 0 ? "-5px" : "",
                                                                                backgroundColor: group.GroupColor
                                                                            }}
                                                                        />
                                                                    </Link>
                                                                )) : null
                                                        }
                                                    </td>
                                            </tr>
                                        ))
                                    }
                                </tbody>
                            </Table>
                        </div> :
                        <div className="align-items-center d-flex flex-column h-100 justify-content-center">
                                <p className="text-muted">You don't have any departments yet. Click the "Add Department" button to add your first department</p>
                                <Link to="/departments">
                                    <Button><BsPlus /> Add Department</Button>
                                </Link>
                            </div>
                }

            </Card.Body>
        </Card>
    )
}

export default Departments;
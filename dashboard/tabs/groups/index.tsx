import React from "react";
import { Card, Table, Button, Image } from "react-bootstrap";
import { Link } from "react-router-dom";
import { Link as RouterLink } from "react-router-dom";
import Avatar from "../../../../components/avatar";
import { BsPlus } from "react-icons/bs";

interface GroupsProps {
    groups,
}

const Groups: React.FC<GroupsProps> = (props) => {
    const { groups } = props;
    return (
        <Card className="card-custom flex-grow-1 h-100 overflow-auto custom-scrollbar">
            <Card.Body className="p-5 border-bottom-0">
                {
                    groups?.length
                        ?
                        <div className="table-responsive">
                            <Table className="user-table" responsive>
                                <thead>
                                    <tr>
                                        <th>Group</th>
                                        <th>Integrations</th>
                                        <th>People</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {
                                        groups.map((group, idx) => (
                                            <tr key={idx}>
                                                <td>
                                                    <RouterLink to={`/groups/${group.GroupID}`} >
                                                        <Avatar
                                                            name={group?.GroupName}
                                                            style={{ backgroundColor: group.GroupColor }}
                                                        />
                                                        <span className="ml-3">{group?.GroupName}</span>
                                                    </RouterLink>
                                                </td>
                                                <td>
                                                    {
                                                        group.GroupIntegrations != undefined ?
                                                        group.GroupIntegrations.map((integrations, index) => (
                                                            <Image key={index} src={integrations.DisplayPhoto} className="mr-3 integration__avatar" />
                                                        )) : <span></span>
                                                    }
                                                </td>
                                                <td>
                                                    {
                                                        group.GroupMembers != undefined ?
                                                            group.GroupMembers.map((member, idx) => {
                                                                return (
                                                                    <Link
                                                                        key={idx}
                                                                        to={`/groups/${member.GroupID}`}
                                                                    >
                                                                        <Avatar
                                                                            name={member.Name}
                                                                            className="avatar--small  "
                                                                            image={member?.DisplayPhoto}
                                                                            style={{
                                                                                marginLeft: idx != 0 ? "-5px" : "",
                                                                            }}
                                                                        />
                                                                    </Link>
                                                                )
                                                            }) : <span></span>
                                                    }
                                                </td>
                                            </tr>
                                        ))
                                    }
                                </tbody>
                            </Table>
                        </div>
                        :
                        <div className="align-items-center d-flex flex-column h-100 justify-content-center">
                            <p className="text-muted">You don't have any groups yet. Click the "Add Group" button to add your first group</p>
                            <Link to="/groups">
                                <Button> <BsPlus/> Add Group</Button>
                            </Link>
                        </div>
                }

            </Card.Body>
        </Card>
    )
}

export default Groups;
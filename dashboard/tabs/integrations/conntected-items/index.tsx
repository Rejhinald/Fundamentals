import React, { useEffect } from "react";
import { Card } from "react-bootstrap";
import { connect } from "react-redux";
import { AppState } from "../../../../../redux/reducers";
import Avatar from "../../../../../components/avatar";
import "./style.scss";
import { Link } from "react-router-dom";

const ConnectedItems = (props) => {
    const {
        integration,
        loading,
        signedInUser
    } = props;

    return (
        <Card className="card-custom h-100">
            <Card.Body className="p-5 d-flex flex-column">
                <div className="d-flex align-items-center mb-10">
                    <Avatar
                        name={integration.IntegrationName}
                        image={integration.DisplayPhoto}
                        className="connected-items-thumbnail"
                        style={{ marginRight: "15px" }}
                    />
                    <Link to={`/integrations/${integration.IntegrationID}`}>
                        <div className="mx-3 mr-2">
                            <p className="font-weight-bolder mb-2" style={{ fontSize: "15px" }}>{integration.IntegrationName}</p>
                            <p className="mb-0 font-size-sm text-muted">{integration.IntegrationDescription}</p>
                        </div>
                    </Link>
                </div>
                <div className="d-flex flex-column mt-auto">
                    <p className="font-weight-bolder mb-0">Applications</p>
                    <div className="d-flex align-items-center">
                        {
                            integration.SubIntegrations && integration.SubIntegrations.map((sub, idx) => (
                                <Avatar
                                    key={idx}
                                    name={sub.IntegrationName}
                                    image={sub.DisplayPhoto}
                                    style={{ marginRight: "10px", height: "auto", maxWidth: "20px", borderRadius: "unset", marginTop: "5px", maxHeight: "20px", objectFit: "contain" }}
                                />
                            ))
                        }
                    </div>
                </div>
            </Card.Body>
        </Card>
    )
}

const mapStateToProps = (state: AppState) => {
    return {
        loading: state.Integrations.loading,
        signedInUser: state.Auth.user
    }
}

const connector = connect(mapStateToProps)

export default connector(ConnectedItems);
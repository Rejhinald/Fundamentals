import React from "react";
import {
    Row,
    Col,
    Card,
    Button,
    Image
} from "react-bootstrap";
import { Link } from "react-router-dom";
import ConnectedItems from "./conntected-items";

import noIntegrationIcon from "../../../../assets/images/empty_state_integration.svg";
import EmptyState from "../../../../components/empty-state";

interface IntegrationsProps {
    integrations,
}

const Integrations: React.FC<IntegrationsProps> = (props) => {
    const { integrations } = props;
    // 
    return (
        <>
            {
                integrations?.length ? (
                    // <div className="overflow-auto">
                        <Row className="mb-10">
                            {
                                integrations.map((item, idx) => (
                                    <Col lg="4" key={idx} className="mb-5">
                                        <ConnectedItems
                                            key={idx}
                                            integration={item}
                                        />
                                    </Col>
                                ))
                            }
                        </Row>

                    // </div>
                ) :
                    <Card className="card-custom h-100">
                        <Card.Body>
                            <div className="align-items-center d-flex flex-column h-100 justify-content-center">

                                <EmptyState
                                    media={<Image src={noIntegrationIcon} alt="no inetgration found" fluid />}
                                    content={
                                        <>
                                            <p>You don't have an integration connected at the moment.</p>
                                            <Link to="/integrations">
                                                <Button>Connect Integration</Button>
                                            </Link>
                                        </>}
                                />
                            </div>
                        </Card.Body>
                    </Card>
            }
        </>
    )
}

export default Integrations;
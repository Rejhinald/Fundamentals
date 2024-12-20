import React from "react";
import { 
    Card,
    Col,
    Row,
} from "react-bootstrap";
import ContentPlaceholder from "../../../../components/content-placeholder";

const EmptyState: React.FC<any> = () => {
    return (
        <Card className="card-custom flex-grow-1 h-100 mt-3">
            <Card.Body>
                <Row>
                    <Col xs="3" className="mb-5">
                        <ContentPlaceholder />
                    </Col>
                </Row>
                <Row className="mb-4">
                    <Col xs="auto">
                        <ContentPlaceholder className="avatar--xs" />
                    </Col>
                    <Col>
                        <Row className="flex-column">
                            <Col xs="6" className="mb-2 px-0">
                                <ContentPlaceholder />
                            </Col>
                            <Col xs="3" className="px-0">
                                <ContentPlaceholder />
                            </Col>
                        </Row>
                    </Col>
                </Row>
                <Row className="mb-4">
                    <Col xs="auto">
                        <ContentPlaceholder className="avatar--xs" />
                    </Col>
                    <Col>
                        <Row className="flex-column">
                            <Col xs="4" className="mb-2 px-0">
                                <ContentPlaceholder />
                            </Col>
                            <Col xs="2" className="px-0">
                                <ContentPlaceholder />
                            </Col>
                        </Row>
                    </Col>
                </Row>
                <Row className="mb-4">
                    <Col xs="auto">
                        <ContentPlaceholder className="avatar--xs" />
                    </Col>
                    <Col>
                        <Row className="flex-column">
                            <Col xs="5" className="mb-2 px-0">
                                <ContentPlaceholder />
                            </Col>
                            <Col xs="3" className="px-0">
                                <ContentPlaceholder />
                            </Col>
                        </Row>
                    </Col>
                </Row>
            </Card.Body>
        </Card>
    );
};

export default EmptyState;
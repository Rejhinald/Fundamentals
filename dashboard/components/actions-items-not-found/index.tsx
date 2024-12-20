import React, { useState } from "react";
import noActionItemsIcon from "../../../../assets/images/action_items.svg";
import { Row, Col } from "react-bootstrap";
import "./style.scss";

const NoActionFound = (props) => {

    return (
        <Col className=".no_action_items__col">
           <div className="d-flex align-items-center flex-column">
                <img src={noActionItemsIcon}  alt="No Action Found" className=".no_action_items__icon" />
                <p className="no_action_items__desc">All actions and recommendations will be listed here.</p>
           </div>
        </Col>
    );
}

export default NoActionFound;

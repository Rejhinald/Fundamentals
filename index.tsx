import React, { useState } from "react";
import { Link, useHistory } from "react-router-dom";
import { Image, Button, DropdownProps } from "react-bootstrap";
import dayjs from "dayjs";
import NotificationPopover from "../popover";
import helper from "../../../../../../../utils/helper";
import { NOTIFICATION_TYPE } from "../../../../../../../utils/constants";
import { useCompanyIntegrations } from "../../../../../../../hooks/Company";
import PermissionsIcon from '../../../../../../../assets/images/sidebar-icon-permissions.svg';
import ActivitiesIcon from '../../../../../../../assets/images/sidebar-icon-activities.svg';
import CompanyIcon from '../../../../../../../assets/images/sidebar-icon-companies.svg'
import IntegrationsIcon from '../../../../../../../assets/images/sidebar-icon-integrations.svg';
import GroupsIcon from '../../../../../../../assets/images/sidebar-icon-groups.svg';
import NotificationRequestModal from "../modals";
interface NotificationItemProps {
  notification;
  seenNotification: (id: string, close?: boolean) => void;
  deleteNotification: (id: string) => void;
  modalRef: React.RefObject<HTMLDivElement>;
  onHideNotification: () => void;
}

interface NotificationInfo {
  title: string;
  icon: string;
}

const NotificationItem: React.FC<NotificationItemProps> = (props) => {
  const { notification, modalRef, onHideNotification } = props;
  const { seenNotification, deleteNotification } = props;
  const [show, setShow] = useState(false)
  const [method, setMethod] = useState(false);
  const companyIntegrations = useCompanyIntegrations()
  const [actionTaken, setActionTaken] = useState(false);
  const [buttonsVisible, setButtonsVisible] = useState(true);

  const companySubIntegrations = companyIntegrations.map(({ SubIntegrations }) => SubIntegrations).flat();
 
  const notificationIntegration = companySubIntegrations?.find(item => item?.IntegrationSlug === notification?.NotificationContent?.NotificationIntegration?.IntegrationSlug)
  const history = useHistory();

  const notificationInfo = (): NotificationInfo => {
    switch (notification?.NotificationType) {
      case "REQUEST_REMOVE_INTEGRATION":
        return {
          title: "Request Integration Removal",
          icon: (notificationIntegration?.DisplayPhoto !== "" ? notificationIntegration?.DisplayPhoto : IntegrationsIcon) || IntegrationsIcon,
        };
        
      case NOTIFICATION_TYPE.REQUEST_CONNECT_INTEGRATION:
        return {
          title: "Request Connect Integration",
          icon: (notificationIntegration?.DisplayPhoto !== "" ? notificationIntegration?.DisplayPhoto : IntegrationsIcon) || IntegrationsIcon,
        };
      case NOTIFICATION_TYPE.REQUEST_DISCONNECT_INTEGRATION:
        return {
          title: "Request Disconnect Integration",
          icon: (notificationIntegration?.DisplayPhoto !== "" ? notificationIntegration?.DisplayPhoto : IntegrationsIcon) || IntegrationsIcon,
        };
      case "REQUEST_PERMISSION_UPDATE":
        return {
          title: "Request Permission Update",
          icon: (notificationIntegration?.DisplayPhoto !== "" ? notificationIntegration?.DisplayPhoto : IntegrationsIcon) || IntegrationsIcon,
        };
      case "REQUEST_COMPANY_ROLE_UPDATE":
        return {
          title: "Request Role Update",
          icon: PermissionsIcon,
        };
      case "REQUEST_TO_JOIN_GROUP":
        return {
          title: "Request To Join Group",
          icon: GroupsIcon,
        };
      case "REQUEST_STATUS_UPDATE":
        return {
          title: "Request Update",
          icon: PermissionsIcon,
        };
      case "REQUEST_TO_CREATE_ACCOUNT":
        return {
          title: "Request to Create/Invite Account",
          icon: (() => {
            const matchedIntegration = companyIntegrations.find((item) => {
              return item.IntegrationSlug === notification.NotificationContent.NotificationIntegration.IntegrationSlug;
            });
      
            return matchedIntegration ? matchedIntegration.DisplayPhoto : IntegrationsIcon;
          })()
        }
      case "REQUEST_TO_MATCH_ACCOUNT":
        return {
          title: "Request to Match Account",
          icon: (() => {
            const matchedIntegration = companyIntegrations.find((item) => {
              return item.IntegrationSlug === notification.NotificationContent.NotificationIntegration.IntegrationSlug;
            });
      
            return matchedIntegration ? matchedIntegration.DisplayPhoto : IntegrationsIcon;
          })()
        }
      case "ROLE_UPDATE": 
        return {
          title: "Company Access Update",
          icon: CompanyIcon
        }
      default:
        return {
          title: "Changes",
          icon: ActivitiesIcon, 
        };
    }
  };
 
  

  const { title, icon } = notificationInfo();
  const handleRequestModal = (method) => {
    setMethod(method);
    setShow(true);
  }
  
  
  const handleActionTaken = () => {
    setActionTaken(true); // Mark the action as taken
    setButtonsVisible(false); // Hide the buttons after action is taken
  }

  const redirectToIntegrationRemoveGroup = () => {
    if (notification?.NotificationType === NOTIFICATION_TYPE.REQUEST_REMOVE_INTEGRATION) {
      onHideNotification();
      history.push(`/integrations/${notification?.NotificationContent?.NotificationIntegration?.IntegrationID}?page=Group&removeGroup=${notification?.NotificationContent?.GroupID}&notification=${notification?.NotificationID}`);
    }
    if (notification?.NotificationType === NOTIFICATION_TYPE.REQUEST_CONNECT_INTEGRATION) {
      onHideNotification();
      history.push(`/integrations/${notification?.NotificationContent?.RequestedIntegration}?action=connect`);
    }
  }
  
  const handleAcceptCreateAccount = () => {
    onHideNotification();
    history.push(`/users/${notification?.NotificationContent?.RequesterUserID}?tab=accounts`);
  }

  // Function to reject the account creation request
  const handleRejectCreateAccount = async () => {
  };
  
  return (
    <li className={`notif flex-column ${!notification.Seen ? "unread" : ""}`}>
      <div className="d-flex notif__content">
        <Image
          src={icon}
          className="notif__photo"
        />
        <div>
          <h4>{notificationIntegration?.IntegrationName}</h4>
          <p className="m-0">
            <span className="font-weight-boldest">
              {title} 
            </span>
            <p className="mb-0">{notification?.NotificationContent?.Message}</p>
          </p>
         
          <p className="font-size-sm text-muted ">
            {dayjs
              .unix(notification?.CreatedAt)
                .format("MMMM DD, YYYY")}
          </p>
        </div>
        <NotificationPopover
          onRead={() => seenNotification(notification.NotificationID, false)}
          onRemove={() => deleteNotification(notification.NotificationID)}
          isSeen={notification?.Seen ?? false}
        />
        
      </div>
      {buttonsVisible && !actionTaken && (
        <>
          {(notification?.NotificationType === 'REQUEST_COMPANY_ROLE_UPDATE' ||
            notification?.NotificationType === 'REQUEST_TO_JOIN_GROUP'  ||
            notification?.NotificationType === NOTIFICATION_TYPE.REQUEST_DISCONNECT_INTEGRATION) && !['ACCEPTED', 'REJECTED'].includes(notification?.NotificationContent?.IsAccepted) && (
              <div className="d-flex justify-content-end">
                <Button variant="link" className="px-5 py-2 rounded" onClick={() => handleRequestModal(false)}>Reject</Button>
                <Button variant="primary" className="px-5 py-2 rounded" onClick={() => handleRequestModal(true)}>Accept</Button>
              </div>
            )}
          {(notification?.NotificationType === NOTIFICATION_TYPE.REQUEST_CONNECT_INTEGRATION ||
            notification?.NotificationType === NOTIFICATION_TYPE.REQUEST_REMOVE_INTEGRATION) && !['ACCEPTED', 'REJECTED'].includes(notification?.NotificationContent?.IsAccepted) && (
              <div className="d-flex justify-content-end">
                <Button variant="link" className="px-5 py-2 rounded" onClick={() => handleRequestModal(false)}>Reject</Button>
                <Button variant="primary" className="px-5 py-2 rounded" onClick={redirectToIntegrationRemoveGroup}>Go to Integration</Button>
              </div>
            )}
          {notification?.NotificationType === 'REQUEST_TO_CREATE_ACCOUNT' && (
            <div className="d-flex justify-content-end">
              <Button variant="link" className="px-5 py-2 rounded" onClick={handleRejectCreateAccount}>Reject</Button>
              <Button variant="primary" className="px-5 py-2 rounded" onClick={handleAcceptCreateAccount}>Accept</Button>
            </div>
          )}
        </>
      )}

     

      

      {/* <div className="text-right">
        <Link
          to={{
            pathname: "/integrations",
            search: `?connect=${helper.toKebabCase(
              notification.IntegrationName
            )}`,
          }}
          onClick={() => seenNotification(notification.NotificationID)}
        >
          <Button>Connect</Button>
        </Link>
      </div> */}
      <NotificationRequestModal 
        show={show} 
        onHide={setShow} 
        requestMethod={method} 
        requestType={notification?.NotificationType}
        modalRef={modalRef}
        notification={notification}
        onActionTaken={handleActionTaken}
      />
    </li>
  );
};

export default NotificationItem;

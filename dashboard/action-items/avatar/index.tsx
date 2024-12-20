import React from 'react';
import Avatar from '../../../../components/avatar';

const ActionAvatar = ({companyData}) => {
    if(!companyData?.CompanyName) return <></>
    
    return (
        <Avatar
            name={companyData?.CompanyName}
            image={companyData?.DisplayPhoto}
            className="avatar--small  "
        />
    );
}

export default ActionAvatar;
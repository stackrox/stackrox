import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { ToastContainer, toast } from 'react-toastify';

import { selectors } from 'reducers';

function Notifications(): ReactElement {
    const notifications = useSelector(selectors.notificationsSelector);

    return (
        <ToastContainer
            toastClassName="toast-selector bg-base-100"
            hideProgressBar
            autoClose={8000}
        >
            {notifications.length !== 0 && toast(notifications[0])}
        </ToastContainer>
    );
}

export default Notifications;

import React from 'react';
import type { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { ToastContainer, toast } from 'react-toastify';

import { selectors } from 'reducers';

function Notifications(): ReactElement {
    const notifications = useSelector(selectors.notificationsSelector);

    return (
        /*
        (dv 2024-05-01)
        Upgrading to React types 18 causes a type error below due to the `children` prop being removed from the `React.FC` type

        @ts-expect-error ToastContainer does not expect children as a prop */
        <ToastContainer toastClassName="toast-selector" hideProgressBar autoClose={8000}>
            {notifications.length !== 0 && toast(notifications[0])}
        </ToastContainer>
    );
}

export default Notifications;

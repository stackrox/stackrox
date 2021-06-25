import { useDispatch } from 'react-redux';

import { actions } from 'reducers/notifications';

const useNotifications = (): ((message) => void) => {
    const dispatch = useDispatch();

    function addNotification(message) {
        dispatch(actions.addNotification(message));
        setTimeout(dispatch(actions.removeOldestNotification), 5000);
    }

    return addNotification;
};

export default useNotifications;

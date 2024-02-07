import { useState } from 'react';

export type AlertObj = {
    type: 'success' | 'danger' | 'warning' | 'info' | 'default';
    title: string;
    children?: React.ReactNode; // inclusive of ReactElement | ReactFragment, or primitives like string or number
};

function useAlert() {
    const [alertObj, setAlertObj] = useState<AlertObj | null>(null);

    function clearAlertObj() {
        setAlertObj(null);
    }

    return { alertObj, setAlertObj, clearAlertObj };
}

export default useAlert;

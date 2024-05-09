import { AlertProps } from '@patternfly/react-core';
import { ReactNode, useState } from 'react';

export type AlertVariantType = AlertProps['variant'];

export type Toast = {
    title: string;
    variant: AlertVariantType;
    key: string;
    children?: ReactNode;
};

type UseToasts = {
    toasts: Toast[];
    addToast: (title: string, variant?: AlertVariantType, children?: ReactNode) => void;
    removeToast: (key: string) => void;
};

function useToasts(): UseToasts {
    const [toasts, setToasts] = useState<Toast[]>([]);

    function getUniqueId() {
        return `${new Date().toISOString()} ${Math.random()}`;
    }

    function addToast(title: string, variant: AlertVariantType = undefined, children: ReactNode) {
        const key = getUniqueId();
        setToasts((prevToasts) => [{ title, variant, key, children }, ...prevToasts]);
    }

    function removeToast(key: string) {
        setToasts((prevToasts) => [...prevToasts.filter((el) => el.key !== key)]);
    }

    return {
        toasts,
        addToast,
        removeToast,
    };
}

export default useToasts;

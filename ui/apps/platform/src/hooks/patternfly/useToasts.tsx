import { ReactNode, useState } from 'react';

export type AlertVariantType = 'default' | 'info' | 'success' | 'danger' | 'warning';

export type Toast = {
    title: string;
    variant: AlertVariantType;
    key: string;
    children?: ReactNode;
};

type UseToasts = {
    toasts: Toast[];
    addToast: (title, variant?: AlertVariantType, children?: ReactNode) => void;
    removeToast: (key) => void;
};

function useToasts(): UseToasts {
    const [toasts, setToasts] = useState<Toast[]>([]);

    function getUniqueId() {
        return `${new Date().toISOString()} ${Math.random()}`;
    }

    function addToast(title, variant = 'default' as AlertVariantType, children) {
        const key = getUniqueId();
        setToasts((prevToasts) => [{ title, variant, key, children }, ...prevToasts]);
    }

    function removeToast(key) {
        setToasts((prevToasts) => [...prevToasts.filter((el) => el.key !== key)]);
    }

    return {
        toasts,
        addToast,
        removeToast,
    };
}

export default useToasts;

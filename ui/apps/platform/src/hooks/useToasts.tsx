import { ReactElement, useState } from 'react';

type AlertVariantType = 'default' | 'info' | 'success' | 'danger' | 'warning';

type Toast = {
    title: string;
    variant: AlertVariantType;
    key: number;
    children?: ReactElement;
};

type UseToasts = {
    toasts: Toast[];
    addToast: (title, variant?: AlertVariantType, children?: ReactElement) => void;
    removeToast: (key) => void;
};

function useToasts(): UseToasts {
    const [toasts, setToasts] = useState<Toast[]>([]);

    function getUniqueId() {
        return new Date().getTime();
    }

    function addToast(title, variant = 'default' as AlertVariantType, children) {
        const key = getUniqueId();
        setToasts([...toasts, { title, variant, key, children }]);
    }

    function removeToast(key) {
        setToasts([...toasts.filter((el) => el.key !== key)]);
    }

    return {
        toasts,
        addToast,
        removeToast,
    };
}

export default useToasts;

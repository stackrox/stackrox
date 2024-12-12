import { useEffect, useCallback } from 'react';

const useClickOutside = (
    ref: React.RefObject<HTMLElement>,
    callback: () => void,
    isOpen: boolean
) => {
    const handleClickOutside = useCallback(
        (event: MouseEvent) => {
            const { target } = event;
            if (ref.current && target instanceof HTMLElement && !ref.current.contains(target)) {
                callback();
            }
        },
        [callback, ref]
    );

    useEffect(() => {
        if (isOpen) {
            document.addEventListener('mousedown', handleClickOutside);
        }

        return () => {
            if (isOpen) {
                document.removeEventListener('mousedown', handleClickOutside);
            }
        };
    }, [isOpen, handleClickOutside]);
};

export default useClickOutside;

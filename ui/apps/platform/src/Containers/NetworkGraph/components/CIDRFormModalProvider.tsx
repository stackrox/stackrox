import React, { createContext, useContext, useState } from 'react';

const defaultValue = {
    isCIDRFormModalOpen: false,
    initialCIDRFormValue: '',
    toggleCIDRFormModal: () => {},
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    setInitialCIDRFormValue: (value: string) => {},
};

const CIDRFormModalContext = createContext(defaultValue);

export const CIDRFormModalProvider = ({ children }) => {
    const [isCIDRFormModalOpen, setIsCIDRFormModalOpen] = useState(false);
    const [initialCIDRFormValue, setInitialCIDRFormValue] = useState<string>('');

    const toggleCIDRFormModal = () => {
        setIsCIDRFormModalOpen((prevValue) => !prevValue);
    };

    return (
        <CIDRFormModalContext.Provider
            value={{
                isCIDRFormModalOpen,
                initialCIDRFormValue,
                setInitialCIDRFormValue,
                toggleCIDRFormModal,
            }}
        >
            {children}
        </CIDRFormModalContext.Provider>
    );
};

export const useCIDRFormModal = () => useContext(CIDRFormModalContext);

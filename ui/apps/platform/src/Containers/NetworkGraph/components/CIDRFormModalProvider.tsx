import React, { createContext, Dispatch, SetStateAction, useContext, useState } from 'react';

interface CIDRFormModalContextType {
    isCIDRFormModalOpen: boolean;
    initialCIDRFormValue: string;
    toggleCIDRFormModal: () => void;
    setInitialCIDRFormValue: Dispatch<SetStateAction<string>>;
}

const defaultValue = {
    isCIDRFormModalOpen: false,
    initialCIDRFormValue: '',
    toggleCIDRFormModal: () => {},
    setInitialCIDRFormValue: () => {},
};

const CIDRFormModalContext = createContext<CIDRFormModalContextType>(defaultValue);

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

import React from 'react';
import { Modal } from '@patternfly/react-core';

import useLocalStorage from 'hooks/useLocalStorage';

const emptyStorage = {
    preferences: {
        defaultFilters: {},
    },
};

function DefaultFilterModal() {
    const [storedValue, setStoredValue] = useLocalStorage('vulnerabilityManagement', emptyStorage);
    return <div>Default vulnerability filters</div>;
}

export default DefaultFilterModal;

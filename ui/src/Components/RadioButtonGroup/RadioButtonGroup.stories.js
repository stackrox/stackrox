import React, { useState } from 'react';
import { MemoryRouter } from 'react-router-dom';

import RadioButtonGroup from './RadioButtonGroup';

export default {
    title: 'RadioButtonGroup',
    component: RadioButtonGroup
};

export const withButtons = () => {
    const buttons = [{ text: 'Fixable' }, { text: 'All' }];
    function onClick() {}
    return (
        <MemoryRouter>
            <RadioButtonGroup headerText="Filter CVEs" buttons={buttons} onClick={onClick} />
        </MemoryRouter>
    );
};

export const withDefaultSelected = () => {
    const buttons = [{ text: 'Fixable' }, { text: 'All' }];
    const selected = 'All';
    function onClick() {}
    return (
        <MemoryRouter>
            <RadioButtonGroup
                headerText="Filter CVEs"
                buttons={buttons}
                selected={selected}
                onClick={onClick}
            />
        </MemoryRouter>
    );
};

export const withSelectableButtons = () => {
    // eslint-disable-next-line
    const [selected, setSelected] = useState('All');
    const buttons = [{ text: 'Fixable' }, { text: 'All' }];
    function onClick(data) {
        setSelected(data);
    }
    return (
        <MemoryRouter>
            <RadioButtonGroup
                headerText="Filter CVEs"
                buttons={buttons}
                selected={selected}
                onClick={onClick}
            />
        </MemoryRouter>
    );
};

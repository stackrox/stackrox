import React, { useState } from 'react';
import { Meta, Story } from '@storybook/react/types-6-0';
import { Lock, Unlock } from 'react-feather';

import ToggleButtonGroup from './ToggleButtonGroup';
import ToggleButton from '../ToggleButton';

export default {
    title: 'ToggleButtonGroup',
    component: ToggleButtonGroup,
} as Meta;

const VALUE = {
    LOCK: 'LOCK',
    UNLOCK: 'UNLOCK',
};

export const BasicUsage: Story = () => {
    const [activeToggleButton, setActiveToggleButton] = useState(VALUE.LOCK);
    return (
        <ToggleButtonGroup activeToggleButton={activeToggleButton}>
            <ToggleButton value={VALUE.LOCK} text="Lock" onClick={setActiveToggleButton} />
            <ToggleButton value={VALUE.UNLOCK} text="Unlock" onClick={setActiveToggleButton} />
        </ToggleButtonGroup>
    );
};

export const WithIcons: Story = () => {
    const [activeToggleButton, setActiveToggleButton] = useState(VALUE.LOCK);
    return (
        <ToggleButtonGroup activeToggleButton={activeToggleButton}>
            <ToggleButton
                icon={Lock}
                value={VALUE.LOCK}
                text="Lock"
                onClick={setActiveToggleButton}
            />
            <ToggleButton
                icon={Unlock}
                value={VALUE.UNLOCK}
                text="Unlock"
                onClick={setActiveToggleButton}
            />
        </ToggleButtonGroup>
    );
};

import React, { useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import ToggleButtonGroup from './ToggleButtonGroup';
import ToggleButton from '../ToggleButton';

function Component() {
    const VALUE = {
        LOCK: 'LOCK',
        UNLOCK: 'UNLOCK',
    };
    const [activeToggleButton, setActiveToggleButton] = useState(VALUE.LOCK);
    return (
        <ToggleButtonGroup activeToggleButton={activeToggleButton}>
            <ToggleButton value={VALUE.LOCK} text="Lock" onClick={setActiveToggleButton} />
            <ToggleButton value={VALUE.UNLOCK} text="Unlock" onClick={setActiveToggleButton} />
        </ToggleButtonGroup>
    );
}

describe('ToggleButtonGroup', () => {
    test('renders the toggle buttons correctly', () => {
        const { container } = render(<Component />);

        expect(container).toMatchSnapshot();
    });

    test('toggles the active button when clicking the inactive button', async () => {
        render(<Component />);

        expect(screen.getByTestId('active-toggle-button')).toHaveTextContent('Lock');

        await userEvent.click(screen.getByText('Unlock'));

        expect(screen.getByTestId('active-toggle-button')).toHaveTextContent('Unlock');
    });
});

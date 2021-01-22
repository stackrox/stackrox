import React, { useState } from 'react';
import { render, fireEvent } from '@testing-library/react';

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

    test('toggles the active button when clicking the inactive button', () => {
        const { getByText, getByTestId } = render(<Component />);

        expect(getByTestId('active-toggle-button')).toHaveTextContent('Lock');

        fireEvent(
            getByText('Unlock'),
            new MouseEvent('click', {
                bubbles: true,
                cancelable: true,
            })
        );

        expect(getByTestId('active-toggle-button')).toHaveTextContent('Unlock');
    });
});

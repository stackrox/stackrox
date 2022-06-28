import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import HelpIcon from './HelpIcon';

describe('HelpIcon', () => {
    test('show the help description', async () => {
        render(<HelpIcon description="Remember to wash your hands" />);

        expect(screen.queryByText('Remember to wash your hands')).not.toBeInTheDocument();

        await userEvent.hover(screen.getByTestId('help-icon'));

        expect(screen.getByText('Remember to wash your hands')).toBeInTheDocument();
    });
});

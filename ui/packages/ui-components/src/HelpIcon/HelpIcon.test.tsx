import React from 'react';
import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import HelpIcon from './HelpIcon';

describe('HelpIcon', () => {
    test('show the help description', () => {
        const { queryByText, getByTestId } = render(
            <HelpIcon description="Remember to wash your hands" />
        );

        expect(queryByText('Remember to wash your hands')).not.toBeInTheDocument();

        userEvent.hover(getByTestId('help-icon'));

        expect(queryByText('Remember to wash your hands')).toBeInTheDocument();
    });
});

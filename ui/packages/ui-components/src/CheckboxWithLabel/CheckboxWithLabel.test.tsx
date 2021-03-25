import React from 'react';
import { render } from '@testing-library/react';

import CheckboxWithLabel from './CheckboxWithLabel';

function doNothing() {}

describe('CheckboxWithLabel', () => {
    test('should be checked by default', () => {
        const { getByLabelText } = render(
            <CheckboxWithLabel
                id="checkbox"
                ariaLabel="This is checked"
                checked
                onChange={doNothing}
            >
                This is checked
            </CheckboxWithLabel>
        );
        const checkbox = getByLabelText('This is checked') as HTMLInputElement;
        expect(checkbox.checked).toEqual(true);
    });

    test('should be unchecked by default', () => {
        const { getByLabelText } = render(
            <CheckboxWithLabel
                id="checkbox"
                ariaLabel="This is checked"
                checked={false}
                onChange={doNothing}
            >
                This is unchecked
            </CheckboxWithLabel>
        );
        const checkbox = getByLabelText('This is unchecked') as HTMLInputElement;
        expect(checkbox.checked).toEqual(false);
    });
});

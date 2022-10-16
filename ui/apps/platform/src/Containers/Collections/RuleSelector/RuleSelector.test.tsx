/* eslint-disable @typescript-eslint/no-non-null-assertion */
import React, { useEffect, useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import RuleSelector from './RuleSelector';
import { ScopedResourceSelector } from '../types';

jest.setTimeout(10000);

// Component wrapper to allow a higher level component to feed updated state back to the RuleSelector.
function DeploymentRuleSelector({ defaultSelector, onChange }) {
    const [resourceSelector, setResourceSelector] = useState<ScopedResourceSelector | null>(
        defaultSelector
    );

    useEffect(() => {
        onChange(resourceSelector);
    }, [resourceSelector, onChange]);

    return (
        <RuleSelector
            entityType="Deployment"
            scopedResourceSelector={resourceSelector}
            handleChange={(_, newSelector) => setResourceSelector(newSelector)}
        />
    );
}

describe('Collection RuleSelector component', () => {
    it('Should render "All entities" option when selector is null', async () => {
        let resourceSelector: ScopedResourceSelector | null = null;

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={resourceSelector} onChange={onChange} />);

        expect(await screen.findByText('All deployments')).toBeInTheDocument();
    });

    it('Should allow users to add name selectors', async () => {
        let resourceSelector: ScopedResourceSelector | null = null;

        const user = userEvent.setup();

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={resourceSelector} onChange={onChange} />);

        await user.click(screen.getByLabelText('Select deployments by name or label'));
        await user.click(screen.getByText('Deployments with names matching'));

        expect(resourceSelector).not.toBeNull();
        expect(resourceSelector!.field).toBe('Deployment');
        expect(resourceSelector!.rules).toEqual([{ operator: 'OR', values: [{ value: '' }] }]);

        const typeAheadInput = screen.getByLabelText('Select a value for the deployment name');
        await user.type(typeAheadInput, 'visa-processor{Enter}');

        expect(resourceSelector!.field).toBe('Deployment');
        expect(resourceSelector!.rules).toEqual([
            { operator: 'OR', values: [{ value: 'visa-processor' }] },
        ]);
        expect(typeAheadInput).toHaveValue('visa-processor');

        // Attempt to add multiple blank values
        await user.click(screen.getByText('Add value'));
        await user.click(screen.getByText('Add value'));
        await user.click(screen.getByText('Add value'));

        // Only a single blank value should be added
        expect(resourceSelector!.rules).toEqual([
            { operator: 'OR', values: [{ value: 'visa-processor' }, { value: '' }] },
        ]);

        // Add a couple more values
        await user.type(
            screen.getAllByLabelText('Select a value for the deployment name')[1],
            'mastercard-processor{Enter}'
        );
        await user.click(screen.getByText('Add value'));
        await user.type(
            screen.getAllByLabelText('Select a value for the deployment name')[2],
            'discover-processor{Enter}'
        );

        expect(resourceSelector!.rules).toEqual([
            {
                operator: 'OR',
                values: [
                    { value: 'visa-processor' },
                    { value: 'mastercard-processor' },
                    { value: 'discover-processor' },
                ],
            },
        ]);

        await user.click(screen.getByLabelText('Delete mastercard-processor'));

        // Check that deletion in the center works
        expect(resourceSelector!.rules).toEqual([
            {
                operator: 'OR',
                values: [{ value: 'visa-processor' }, { value: 'discover-processor' }],
            },
        ]);

        // Check that deletion of all items removes the selector
        await user.click(screen.getByLabelText('Delete visa-processor'));
        await user.click(screen.getByLabelText('Delete discover-processor'));

        expect(resourceSelector).toBeNull();
        expect(screen.getByText('All deployments')).toBeInTheDocument();
    });

    it('Should allow users to add label selectors', async () => {});
});

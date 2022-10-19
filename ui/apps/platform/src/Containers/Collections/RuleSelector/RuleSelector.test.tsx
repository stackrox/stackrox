import React, { useEffect, useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import RuleSelector from './RuleSelector';
import { ByLabelResourceSelector, ByNameResourceSelector, ScopedResourceSelector } from '../types';

jest.setTimeout(10000);

// Component wrapper to allow a higher level component to feed updated state back to the RuleSelector.
function DeploymentRuleSelector({ defaultSelector, onChange }) {
    const [resourceSelector, setResourceSelector] =
        useState<ScopedResourceSelector>(defaultSelector);

    useEffect(() => {
        onChange(resourceSelector);
    }, [resourceSelector, onChange]);

    return (
        <RuleSelector
            entityType="Deployment"
            scopedResourceSelector={resourceSelector}
            handleChange={(_, newSelector) => setResourceSelector(newSelector)}
            validationErrors={undefined}
        />
    );
}

describe('Collection RuleSelector component', () => {
    it('Should render "All entities" option when selector is null', async () => {
        let resourceSelector: ScopedResourceSelector = {};

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={resourceSelector} onChange={onChange} />);

        expect(await screen.findByText('All deployments')).toBeInTheDocument();
    });

    it('Should allow users to add name selectors', async () => {
        let resourceSelector: ByNameResourceSelector = {
            field: 'Deployment',
            rule: { clientId: '', operator: 'OR', values: [] },
        };

        const user = userEvent.setup();

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={{}} onChange={onChange} />);

        await user.click(screen.getByLabelText('Select deployments by name or label'));
        await user.click(screen.getByText('Deployments with names matching'));

        expect(resourceSelector.field).toBe('Deployment');
        expect(resourceSelector.rule.values.map(({ value }) => value)).toEqual(['']);

        const typeAheadInput = screen.getByLabelText('Select a value for the deployment name');
        await user.type(typeAheadInput, 'visa-processor{Enter}');

        expect(resourceSelector.field).toBe('Deployment');
        expect(resourceSelector.rule.values.map(({ value }) => value)).toEqual(['visa-processor']);
        expect(typeAheadInput).toHaveValue('visa-processor');

        // Attempt to add multiple blank values
        await user.click(screen.getByText('Add value'));
        await user.click(screen.getByText('Add value'));

        // Only a single blank value should be added
        expect(resourceSelector.rule.values.map(({ value }) => value)).toEqual([
            'visa-processor',
            '',
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

        expect(resourceSelector.rule.values.map(({ value }) => value)).toEqual([
            'visa-processor',
            'mastercard-processor',
            'discover-processor',
        ]);

        await user.click(screen.getByLabelText('Delete mastercard-processor'));

        // Check that deletion in the center works
        expect(resourceSelector.rule.values.map(({ value }) => value)).toEqual([
            'visa-processor',
            'discover-processor',
        ]);

        // Check that deletion of all items removes the selector
        await user.click(screen.getByLabelText('Delete visa-processor'));
        await user.click(screen.getByLabelText('Delete discover-processor'));

        expect(resourceSelector).toEqual({});
        expect(screen.getByText('All deployments')).toBeInTheDocument();
    });

    it('Should allow users to add label key/value selectors', async () => {
        let resourceSelector: ByLabelResourceSelector = {
            field: 'Deployment Label',
            rules: [
                { clientId: '', operator: 'OR', key: '', values: [{ clientId: '', value: '' }] },
            ],
        };

        const user = userEvent.setup();

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={{}} onChange={onChange} />);

        await user.click(screen.getByLabelText('Select deployments by name or label'));
        await user.click(screen.getByText('Deployments with labels matching'));

        expect(resourceSelector.field).toBe('Deployment Label');
        expect(resourceSelector.rules[0].key).toEqual('');
        expect(resourceSelector.rules[0].values.map(({ value }) => value)).toEqual(['']);

        await user.type(
            screen.getByLabelText('Select a value for the deployment label key'),
            'kubernetes.io/metadata.name{Enter}'
        );
        await user.type(
            screen.getByLabelText('Select a value for the deployment label value'),
            'visa-processor{Enter}'
        );
        expect(resourceSelector.rules[0].key).toEqual('kubernetes.io/metadata.name');
        expect(resourceSelector.rules[0].values.map(({ value }) => value)).toEqual([
            'visa-processor',
        ]);

        // Attempt to add multiple blank values
        await user.click(screen.getByText('Add value'));
        await user.click(screen.getByText('Add value'));

        // Only a single blank value should be added
        expect(resourceSelector.rules[0].values.map(({ value }) => value)).toEqual([
            'visa-processor',
            '',
        ]);

        await user.type(
            screen.getAllByLabelText('Select a value for the deployment label value')[1],
            'mastercard-processor{Enter}'
        );
        await user.click(screen.getByText('Add value'));
        await user.type(
            screen.getAllByLabelText('Select a value for the deployment label value')[2],
            'discover-processor{Enter}'
        );

        expect(resourceSelector.rules[0].values.map(({ value }) => value)).toEqual([
            'visa-processor',
            'mastercard-processor',
            'discover-processor',
        ]);

        // Add another label rule and key'values
        await user.click(screen.getByText('Add label rule'));

        await user.type(
            screen.getAllByLabelText('Select a value for the deployment label key')[1],
            'kubernetes.io/metadata.release{Enter}'
        );
        await user.type(
            screen.getAllByLabelText('Select a value for the deployment label value')[3],
            // typo
            'stabl{Enter}'
        );
        await user.click(screen.getAllByText('Add value')[1]);
        await user.type(
            screen.getAllByLabelText('Select a value for the deployment label value')[4],
            'beta{Enter}'
        );
        // test editing typo
        await user.type(
            screen.getAllByLabelText('Select a value for the deployment label value')[3],
            'e{Enter}'
        );

        expect(resourceSelector.rules[0].values.map(({ value }) => value)).toEqual([
            'visa-processor',
            'mastercard-processor',
            'discover-processor',
        ]);
        expect(resourceSelector.rules[1].values.map(({ value }) => value)).toEqual([
            'stable',
            'beta',
        ]);
    });
});

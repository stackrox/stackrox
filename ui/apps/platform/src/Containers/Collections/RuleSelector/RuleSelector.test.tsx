import React, { useEffect, useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/extend-expect';

import { mockDebounce } from 'test-utils/mocks/@patternfly/react-core';
import RuleSelector from './RuleSelector';
import { ByLabelResourceSelector, ByNameResourceSelector, ScopedResourceSelector } from '../types';

jest.mock('@patternfly/react-core', () => mockDebounce);

jest.mock('services/CollectionsService', () => ({
    __esModule: true,
    getCollectionAutoComplete: () => ({ request: Promise.resolve([]) }),
}));

// Component wrapper to allow a higher level component to feed updated state back to the RuleSelector.
function DeploymentRuleSelector({ defaultSelector, onChange }) {
    const [resourceSelector, setResourceSelector] =
        useState<ScopedResourceSelector>(defaultSelector);

    useEffect(() => {
        onChange(resourceSelector);
    }, [resourceSelector, onChange]);

    return (
        <RuleSelector
            collection={{
                name: '',
                description: '',
                resourceSelector: {
                    Deployment: { type: 'All' },
                    Namespace: { type: 'All' },
                    Cluster: { type: 'All' },
                },
                embeddedCollectionIds: [],
            }}
            entityType="Deployment"
            scopedResourceSelector={resourceSelector}
            handleChange={(_, newSelector) => setResourceSelector(newSelector)}
            validationErrors={undefined}
        />
    );
}

describe('Collection RuleSelector component', () => {
    it('Should render "All entities" option when selector is null', async () => {
        let resourceSelector: ScopedResourceSelector = { type: 'All' };

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={resourceSelector} onChange={onChange} />);

        expect(await screen.findByText('All deployments')).toBeInTheDocument();
    });

    it('Should allow users to add name selectors', async () => {
        let resourceSelector: ByNameResourceSelector = {
            type: 'ByName',
            field: 'Deployment',
            rule: { operator: 'OR', values: [] },
        };

        const user = userEvent.setup();

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={{ type: 'All' }} onChange={onChange} />);

        await user.click(screen.getByLabelText('Select deployments by name or label'));
        await user.click(screen.getByText('Deployments with names matching'));

        expect(resourceSelector.field).toBe('Deployment');
        expect(resourceSelector.rule.values).toEqual(['']);

        const typeAheadInput = screen.getByLabelText('Select value 1 of 1 for the deployment name');
        await user.type(typeAheadInput, 'visa-processor{Enter}');

        expect(resourceSelector.field).toBe('Deployment');
        expect(resourceSelector.rule.values).toEqual(['visa-processor']);
        expect(typeAheadInput).toHaveValue('visa-processor');

        // Attempt to add multiple blank values
        await user.click(screen.getByText('Add value'));
        await user.click(screen.getByText('Add value'));

        // Only a single blank value should be added
        expect(resourceSelector.rule.values).toEqual(['visa-processor', '']);

        // Add a couple more values
        await user.type(
            screen.getByLabelText('Select value 2 of 2 for the deployment name'),
            'mastercard-processor{Enter}'
        );
        await user.click(screen.getByText('Add value'));
        await user.type(
            screen.getByLabelText('Select value 3 of 3 for the deployment name'),
            'discover-processor{Enter}'
        );

        expect(resourceSelector.rule.values).toEqual([
            'visa-processor',
            'mastercard-processor',
            'discover-processor',
        ]);

        await user.click(screen.getByLabelText('Delete mastercard-processor'));

        // Check that deletion in the center works
        expect(resourceSelector.rule.values).toEqual(['visa-processor', 'discover-processor']);

        // Check that deletion of all items removes the selector
        await user.click(screen.getByLabelText('Delete visa-processor'));
        await user.click(screen.getByLabelText('Delete discover-processor'));

        expect(resourceSelector).toEqual({ type: 'All' });
        expect(screen.getByText('All deployments')).toBeInTheDocument();
    });

    it('Should allow users to add label key/value selectors', async () => {
        let resourceSelector: ByLabelResourceSelector = {
            type: 'ByLabel',
            field: 'Deployment Label',
            rules: [{ operator: 'OR', key: '', values: [''] }],
        };

        const user = userEvent.setup();

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={{ type: 'All' }} onChange={onChange} />);

        await user.click(screen.getByLabelText('Select deployments by name or label'));
        await user.click(screen.getByText('Deployments with labels matching'));

        expect(resourceSelector.field).toBe('Deployment Label');
        expect(resourceSelector.rules[0].key).toEqual('');
        expect(resourceSelector.rules[0].values).toEqual(['']);

        await user.type(
            screen.getByLabelText('Select label key for deployment rule 1 of 1'),
            'kubernetes.io/metadata.name{Enter}'
        );
        await user.type(
            screen.getByLabelText('Select label value 1 of 1 for deployment rule 1 of 1'),
            'visa-processor{Enter}'
        );
        expect(resourceSelector.rules[0].key).toEqual('kubernetes.io/metadata.name');
        expect(resourceSelector.rules[0].values).toEqual(['visa-processor']);

        // Attempt to add multiple blank values
        await user.click(screen.getByText('Add value'));
        await user.click(screen.getByText('Add value'));

        // Only a single blank value should be added
        expect(resourceSelector.rules[0].values).toEqual(['visa-processor', '']);

        await user.type(
            screen.getByLabelText('Select label value 2 of 2 for deployment rule 1 of 1'),
            'mastercard-processor{Enter}'
        );
        await user.click(screen.getByText('Add value'));
        await user.type(
            screen.getByLabelText('Select label value 3 of 3 for deployment rule 1 of 1'),
            'discover-processor{Enter}'
        );

        expect(resourceSelector.rules[0].values).toEqual([
            'visa-processor',
            'mastercard-processor',
            'discover-processor',
        ]);

        // Add another label rule and key'values
        await user.click(screen.getByText('Add label rule'));

        await user.type(
            screen.getByLabelText('Select label key for deployment rule 2 of 2'),
            'kubernetes.io/metadata.release{Enter}'
        );
        await user.type(
            screen.getByLabelText('Select label value 1 of 1 for deployment rule 2 of 2'),
            'stable{Enter}'
        );

        await user.click(screen.getAllByText('Add value')[1]);
        await user.type(
            screen.getByLabelText('Select label value 2 of 2 for deployment rule 2 of 2'),
            'beta{Enter}'
        );

        expect(resourceSelector).toEqual({
            type: 'ByLabel',
            field: 'Deployment Label',
            rules: [
                {
                    operator: 'OR',
                    key: 'kubernetes.io/metadata.name',
                    values: ['visa-processor', 'mastercard-processor', 'discover-processor'],
                },
                {
                    operator: 'OR',
                    key: 'kubernetes.io/metadata.release',
                    values: ['stable', 'beta'],
                },
            ],
        });

        // Check that deletion of all items removes the selector
        await user.click(screen.getByLabelText('Delete stable'));
        await user.click(screen.getByLabelText('Delete beta'));
        await user.click(screen.getByLabelText('Delete visa-processor'));
        await user.click(screen.getByLabelText('Delete mastercard-processor'));
        await user.click(screen.getByLabelText('Delete discover-processor'));

        expect(resourceSelector).toEqual({ type: 'All' });
        expect(screen.getByText('All deployments')).toBeInTheDocument();
    });
});

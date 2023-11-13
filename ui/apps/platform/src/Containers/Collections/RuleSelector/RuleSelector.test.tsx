import React, { useEffect, useState } from 'react';
import { render, screen, act } from '@testing-library/react';
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

        await act(() => user.click(screen.getByLabelText('Select deployments by name or label')));
        await act(() => user.click(screen.getByText('Deployments with names matching')));

        expect(resourceSelector.field).toBe('Deployment');
        expect(resourceSelector.rule.values).toEqual([{ value: '', matchType: 'EXACT' }]);

        const typeAheadInput = screen.getByLabelText('Select value 1 of 1 for the deployment name');
        await act(() => user.type(typeAheadInput, 'visa-processor{Enter}'));

        expect(resourceSelector.field).toBe('Deployment');
        expect(resourceSelector.rule.values).toEqual([
            { value: 'visa-processor', matchType: 'EXACT' },
        ]);
        expect(typeAheadInput).toHaveValue('visa-processor');

        // Attempt to add multiple blank values
        await act(() => user.click(screen.getByLabelText('Add deployment name value')));
        await act(() => user.click(screen.getByLabelText('Add deployment name value')));

        // Only a single blank value should be added
        expect(resourceSelector.rule.values).toEqual([
            { value: 'visa-processor', matchType: 'EXACT' },
            { value: '', matchType: 'EXACT' },
        ]);

        // Add a couple more values
        await act(() =>
            user.type(
                screen.getByLabelText('Select value 2 of 2 for the deployment name'),
                'mastercard-processor{Enter}'
            )
        );
        await act(() => user.click(screen.getByLabelText('Add deployment name value')));
        await act(() =>
            user.type(
                screen.getByLabelText('Select value 3 of 3 for the deployment name'),
                'discover-processor{Enter}'
            )
        );

        expect(resourceSelector.rule.values).toEqual([
            { value: 'visa-processor', matchType: 'EXACT' },
            { value: 'mastercard-processor', matchType: 'EXACT' },
            { value: 'discover-processor', matchType: 'EXACT' },
        ]);

        await act(() => user.click(screen.getByLabelText('Delete mastercard-processor')));

        // Check that deletion in the center works
        expect(resourceSelector.rule.values).toEqual([
            { value: 'visa-processor', matchType: 'EXACT' },
            { value: 'discover-processor', matchType: 'EXACT' },
        ]);

        // Check that deletion of all items removes the selector
        await act(() => user.click(screen.getByLabelText('Delete visa-processor')));
        await act(() => user.click(screen.getByLabelText('Delete discover-processor')));

        expect(resourceSelector).toEqual({ type: 'All' });
        expect(screen.getByText('All deployments')).toBeInTheDocument();
    });

    it('Should allow users to add label key/value selectors', async () => {
        let resourceSelector: ByLabelResourceSelector = {
            type: 'ByLabel',
            field: 'Deployment Label',
            rules: [{ operator: 'OR', values: [{ value: '', matchType: 'EXACT' }] }],
        };

        const user = userEvent.setup();

        function onChange(newSelector) {
            resourceSelector = newSelector;
        }

        render(<DeploymentRuleSelector defaultSelector={{ type: 'All' }} onChange={onChange} />);

        await act(() => user.click(screen.getByLabelText('Select deployments by name or label')));
        await act(() => user.click(screen.getByText('Deployments with labels matching exactly')));

        expect(resourceSelector.field).toBe('Deployment Label');
        expect(resourceSelector.rules[0].values).toEqual([{ value: '', matchType: 'EXACT' }]);

        await act(() =>
            user.type(
                screen.getByLabelText('Select label value 1 of 1 for deployment rule 1 of 1'),
                'kubernetes.io/metadata.name=visa-processor{Enter}'
            )
        );
        expect(resourceSelector.rules[0].values).toEqual([
            { value: 'kubernetes.io/metadata.name=visa-processor', matchType: 'EXACT' },
        ]);

        // Attempt to add multiple blank values
        await act(() => user.click(screen.getByLabelText('Add deployment label value for rule 1')));
        await act(() => user.click(screen.getByLabelText('Add deployment label value for rule 1')));

        // Only a single blank value should be added
        expect(resourceSelector.rules[0].values).toEqual([
            { value: 'kubernetes.io/metadata.name=visa-processor', matchType: 'EXACT' },
            { value: '', matchType: 'EXACT' },
        ]);

        await act(() =>
            user.type(
                screen.getByLabelText('Select label value 2 of 2 for deployment rule 1 of 1'),
                'kubernetes.io/metadata.name=mastercard-processor{Enter}'
            )
        );
        await act(() => user.click(screen.getByLabelText('Add deployment label value for rule 1')));
        await act(() =>
            user.type(
                screen.getByLabelText('Select label value 3 of 3 for deployment rule 1 of 1'),
                'kubernetes.io/metadata.name=discover-processor{Enter}'
            )
        );

        expect(resourceSelector.rules[0].values).toEqual([
            { value: 'kubernetes.io/metadata.name=visa-processor', matchType: 'EXACT' },
            { value: 'kubernetes.io/metadata.name=mastercard-processor', matchType: 'EXACT' },
            { value: 'kubernetes.io/metadata.name=discover-processor', matchType: 'EXACT' },
        ]);

        // Add another label rule
        await act(() => user.click(screen.getByText('Add label section (AND)')));

        await act(() =>
            user.type(
                screen.getByLabelText('Select label value 1 of 1 for deployment rule 2 of 2'),
                'kubernetes.io/metadata.release=stable{Enter}'
            )
        );

        await act(() => user.click(screen.getByLabelText('Add deployment label value for rule 2')));
        await act(() =>
            user.type(
                screen.getByLabelText('Select label value 2 of 2 for deployment rule 2 of 2'),
                'kubernetes.io/metadata.release=beta{Enter}'
            )
        );

        expect(resourceSelector).toEqual({
            type: 'ByLabel',
            field: 'Deployment Label',
            rules: [
                {
                    operator: 'OR',
                    values: [
                        {
                            value: 'kubernetes.io/metadata.name=visa-processor',
                            matchType: 'EXACT',
                        },
                        {
                            value: 'kubernetes.io/metadata.name=mastercard-processor',
                            matchType: 'EXACT',
                        },
                        {
                            value: 'kubernetes.io/metadata.name=discover-processor',
                            matchType: 'EXACT',
                        },
                    ],
                },
                {
                    operator: 'OR',
                    values: [
                        { value: 'kubernetes.io/metadata.release=stable', matchType: 'EXACT' },
                        { value: 'kubernetes.io/metadata.release=beta', matchType: 'EXACT' },
                    ],
                },
            ],
        });

        // Check that deletion of all items removes the selector
        await act(() =>
            user.click(screen.getByLabelText('Delete kubernetes.io/metadata.release=stable'))
        );
        await act(() =>
            user.click(screen.getByLabelText('Delete kubernetes.io/metadata.release=beta'))
        );
        await act(() =>
            user.click(screen.getByLabelText('Delete kubernetes.io/metadata.name=visa-processor'))
        );
        await act(() =>
            user.click(
                screen.getByLabelText('Delete kubernetes.io/metadata.name=mastercard-processor')
            )
        );
        await act(() =>
            user.click(
                screen.getByLabelText('Delete kubernetes.io/metadata.name=discover-processor')
            )
        );

        expect(resourceSelector).toEqual({ type: 'All' });
        expect(screen.getByText('All deployments')).toBeInTheDocument();
    });
});

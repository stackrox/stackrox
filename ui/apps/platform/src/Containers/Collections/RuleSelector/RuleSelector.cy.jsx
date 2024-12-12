import React, { useState } from 'react';
import isEqual from 'lodash/isEqual';

import RuleSelector from './RuleSelector';

// Component wrapper to allow a higher level component to feed updated state back to the RuleSelector.
function DeploymentRuleSelector({ defaultSelector, onChange }) {
    const [resourceSelector, setResourceSelector] = useState(defaultSelector);

    return (
        <RuleSelector
            entityType="Deployment"
            scopedResourceSelector={resourceSelector}
            handleChange={(_, newSelector) => {
                setResourceSelector(newSelector);
                onChange(newSelector);
            }}
            validationErrors={undefined}
        />
    );
}

function setup(defaultSelector, onChange) {
    cy.intercept('GET', '/v1/collections/*/autocomplete', (req) => req.reply([]));

    cy.mount(<DeploymentRuleSelector defaultSelector={defaultSelector} onChange={onChange} />);
}

describe(Cypress.spec.relative, () => {
    it('should render "No deployments specified" option when selector is null', () => {
        setup({ type: 'NoneSpecified' }, () => {});

        cy.findByText('No deployments specified');
    });

    it('should allow users to add name selectors', () => {
        const state = {
            resourceSelector: {
                type: 'ByName',
                field: 'Deployment',
                rule: { operator: 'OR', values: [] },
            },
        };

        function onChange(newSelector) {
            state.resourceSelector = newSelector;
        }

        const hasRuleValues = (values) => (s) => isEqual(s.resourceSelector.rule.values, values);

        setup({ type: 'NoneSpecified' }, onChange);

        cy.findByLabelText('Select deployments by name or label').click();
        cy.findByText('Deployments with names matching').click();

        cy.wrap(state).should('deep.equal', {
            resourceSelector: {
                type: 'ByName',
                field: 'Deployment',
                rule: { operator: 'OR', values: [{ value: '', matchType: 'EXACT' }] },
            },
        });

        cy.findByLabelText('Select value 1 of 1 for the deployment name').type(
            'visa-processor{Enter}'
        );

        cy.wrap(state).should(
            'satisfy',
            hasRuleValues([{ value: 'visa-processor', matchType: 'EXACT' }])
        );

        cy.findByLabelText('Select value 1 of 1 for the deployment name').should(
            'have.value',
            'visa-processor'
        );

        // Attempt to add multiple blank values
        cy.findByLabelText('Add deployment name value').click();
        cy.findByLabelText('Add deployment name value').click();

        // Only a single blank value should be added
        cy.wrap(state).should(
            'satisfy',
            hasRuleValues([
                { value: 'visa-processor', matchType: 'EXACT' },
                { value: '', matchType: 'EXACT' },
            ])
        );

        // Add a couple more values
        cy.findByLabelText('Select value 2 of 2 for the deployment name').type(
            'mastercard-processor{Enter}'
        );
        cy.findByLabelText('Add deployment name value').click();
        cy.findByLabelText('Select value 3 of 3 for the deployment name').type(
            'discover-processor{Enter}'
        );

        cy.wrap(state).should(
            'satisfy',
            hasRuleValues([
                { value: 'visa-processor', matchType: 'EXACT' },
                { value: 'mastercard-processor', matchType: 'EXACT' },
                { value: 'discover-processor', matchType: 'EXACT' },
            ])
        );

        cy.findByLabelText('Delete mastercard-processor').click();

        // Check that deletion in the center works
        cy.wrap(state).should(
            'satisfy',
            hasRuleValues([
                { value: 'visa-processor', matchType: 'EXACT' },
                { value: 'discover-processor', matchType: 'EXACT' },
            ])
        );

        // Check that deletion of all items removes the selector
        cy.findByLabelText('Delete visa-processor').click();
        cy.findByLabelText('Delete discover-processor').click();

        cy.wrap(state).should('deep.equal', { resourceSelector: { type: 'NoneSpecified' } });
    });

    it('should allow users to add label key/value selectors', () => {
        const state = {
            resourceSelector: {
                type: 'ByLabel',
                field: 'Deployment Label',
                rules: [{ operator: 'OR', values: [{ value: '', matchType: 'EXACT' }] }],
            },
        };

        function onChange(newSelector) {
            state.resourceSelector = newSelector;
        }

        const hasRuleValues = (values) => (s) =>
            isEqual(s.resourceSelector.rules[0].values, values);

        setup({ type: 'NoneSpecified' }, onChange);

        cy.findByLabelText('Select deployments by name or label').click();
        cy.findByText('Deployments with labels matching exactly').click();

        cy.wrap(state).should('deep.equal', {
            resourceSelector: {
                type: 'ByLabel',
                field: 'Deployment Label',
                rules: [{ operator: 'OR', values: [{ value: '', matchType: 'EXACT' }] }],
            },
        });

        cy.findByLabelText('Select label value 1 of 1 for deployment rule 1 of 1').type(
            'kubernetes.io/metadata.name=visa-processor{Enter}'
        );

        cy.wrap(state).should(
            'satisfy',
            hasRuleValues([
                { value: 'kubernetes.io/metadata.name=visa-processor', matchType: 'EXACT' },
            ])
        );

        // Attempt to add multiple blank values
        cy.findByLabelText('Add deployment label value for rule 1').click();
        cy.findByLabelText('Add deployment label value for rule 1').click();

        // Only a single blank value should be added
        cy.wrap(state).should(
            'satisfy',
            hasRuleValues([
                { value: 'kubernetes.io/metadata.name=visa-processor', matchType: 'EXACT' },
                { value: '', matchType: 'EXACT' },
            ])
        );

        cy.findByLabelText('Select label value 2 of 2 for deployment rule 1 of 1').type(
            'kubernetes.io/metadata.name=mastercard-processor{Enter}'
        );
        cy.findByLabelText('Add deployment label value for rule 1').click();
        cy.findByLabelText('Select label value 3 of 3 for deployment rule 1 of 1').type(
            'kubernetes.io/metadata.name=discover-processor{Enter}'
        );

        cy.wrap(state).should(
            'satisfy',
            hasRuleValues([
                { value: 'kubernetes.io/metadata.name=visa-processor', matchType: 'EXACT' },
                {
                    value: 'kubernetes.io/metadata.name=mastercard-processor',
                    matchType: 'EXACT',
                },
                { value: 'kubernetes.io/metadata.name=discover-processor', matchType: 'EXACT' },
            ])
        );

        // Add another label rule
        cy.findByText('Add label section (AND)').click();

        cy.findByLabelText('Select label value 1 of 1 for deployment rule 2 of 2').type(
            'kubernetes.io/metadata.release=stable{Enter}'
        );
        cy.findByLabelText('Add deployment label value for rule 2').click();
        cy.findByLabelText('Select label value 2 of 2 for deployment rule 2 of 2').type(
            'kubernetes.io/metadata.release=beta{Enter}'
        );

        cy.wrap(state).should('deep.equal', {
            resourceSelector: {
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
            },
        });

        // Check that deletion of all items removes the selector
        cy.findByLabelText('Delete kubernetes.io/metadata.release=stable').click();
        cy.findByLabelText('Delete kubernetes.io/metadata.release=beta').click();
        cy.findByLabelText('Delete kubernetes.io/metadata.name=visa-processor').click();
        cy.findByLabelText('Delete kubernetes.io/metadata.name=mastercard-processor').click();
        cy.findByLabelText('Delete kubernetes.io/metadata.name=discover-processor').click();

        cy.wrap(state).should('deep.equal', { resourceSelector: { type: 'NoneSpecified' } });
    });
});

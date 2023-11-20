import React from 'react';
import { Router } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';
import { createBrowserHistory } from 'history';

import configureApolloClient from 'configureApolloClient';
import { standardEntityTypes } from 'constants/entityTypes';
import { complianceBasePath, urlEntityListTypes } from 'routePaths';
import ComplianceLevelsByStandard from './ComplianceLevelsByStandard';

/*
These standards have been formatted for easier verification of the expected ordering in
tests compared to a direct hard coding in the mocked response below.

id: [name, numFailing, numPassing]
*/
const standards = {
    CIS_Kubernetes_v1_5: ['CIS Kubernetes v1.5', 8, 2],
    HIPAA_164: ['HIPAA 164', 7, 3],
    NIST_800_190: ['NIST SP 800-190', 6, 4],
    NIST_SP_800_53_Rev_4: ['NIST SP 800-53', 5, 5],
    PCI_DSS_3_2: ['PCI DSS 3.2.1', 4, 6],
    'ocp4-cis': ['ocp4-cis', 3, 7],
    'ocp4-cis-node': ['ocp4-cis-node', 2, 8],
};

const mock = {
    data: {
        controls: {
            results: Object.entries(standards).map(([id, [, numFailing, numPassing]]) => ({
                aggregationKeys: [{ id, scope: 'STANDARD' }],
                numFailing,
                numPassing,
                numSkipped: 0,
                unit: 'CONTROL',
            })),
        },
        complianceStandards: Object.entries(standards).map(([id, [name]]) => ({
            id,
            name,
        })),
    },
};

const setup = () => {
    const history = createBrowserHistory();

    cy.intercept('POST', '/api/graphql?opname=getAggregatedResults', (req) => {
        req.reply(mock);
    });

    cy.mount(
        <Router history={history}>
            <ApolloProvider client={configureApolloClient()}>
                <ComplianceLevelsByStandard />
            </ApolloProvider>
        </Router>
    );
};

describe(Cypress.spec.relative, () => {
    it('should render graph bars correctly order by compliance percentage', () => {
        setup();

        const standardNames = Object.values(standards).map(([name]) => name);
        const titlesRegex = new RegExp(`${standardNames.join('|')}`);

        // Default is ascending
        // Note that we use negative indices here because the order in the DOM (bottom->top) is the opposite from
        // how that chart is displayed to the user (top->bottom)
        cy.findAllByText(titlesRegex).eq(-1).should('have.text', 'CIS Kubernetes v1.5');
        cy.findAllByText(titlesRegex).eq(-2).should('have.text', 'HIPAA 164');
        cy.findAllByText(titlesRegex).eq(-3).should('have.text', 'NIST SP 800-190');
        cy.findAllByText(titlesRegex).eq(-4).should('have.text', 'NIST SP 800-53');
        cy.findAllByText(titlesRegex).eq(-5).should('have.text', 'PCI DSS 3.2.1');
        cy.findAllByText(titlesRegex).eq(-6).should('have.text', 'ocp4-cis');

        // Sort by descending
        cy.findByLabelText('Options').click();
        cy.findByText('Descending').click();

        cy.findAllByText(titlesRegex).eq(-1).should('have.text', 'ocp4-cis-node');
        cy.findAllByText(titlesRegex).eq(-2).should('have.text', 'ocp4-cis');
        cy.findAllByText(titlesRegex).eq(-3).should('have.text', 'PCI DSS 3.2.1');
        cy.findAllByText(titlesRegex).eq(-4).should('have.text', 'NIST SP 800-53');
        cy.findAllByText(titlesRegex).eq(-5).should('have.text', 'NIST SP 800-190');
        cy.findAllByText(titlesRegex).eq(-6).should('have.text', 'HIPAA 164');
    });

    it('should visit the correct pages when widget links are clicked', () => {
        setup();

        cy.findByText('View all').click();
        cy.url().should('include', complianceBasePath);

        const standard = 'CIS Kubernetes v1.5';
        cy.findByText(standard).click();
        cy.url().should(
            'include',
            `${complianceBasePath}/${urlEntityListTypes[standardEntityTypes.CONTROL]}`
        );
        cy.url({ decode: true }).should('include', `?s[Cluster]=*&s[standard]=${standard}`);
    });

    it('should contain a button that resets the widget options to default', () => {
        setup();
        cy.findByLabelText('Options').click();

        // Defaults
        cy.findByRole('button', { name: 'Ascending' }).should('have.attr', 'aria-pressed', 'true');
        cy.findByRole('button', { name: 'Descending' }).should(
            'have.attr',
            'aria-pressed',
            'false'
        );

        cy.findByRole('button', { name: 'Descending' }).click();

        cy.findByRole('button', { name: 'Ascending' }).should('have.attr', 'aria-pressed', 'false');
        cy.findByRole('button', { name: 'Descending' }).should('have.attr', 'aria-pressed', 'true');

        cy.findByLabelText('Revert to default options').click();

        // Check return to defaults
        cy.findByLabelText('Options').click();
        cy.findByRole('button', { name: 'Ascending' }).should('have.attr', 'aria-pressed', 'true');
        cy.findByRole('button', { name: 'Descending' }).should(
            'have.attr',
            'aria-pressed',
            'false'
        );
    });
});

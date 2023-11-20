import React from 'react';
import { Router } from 'react-router-dom';
import { ApolloProvider } from '@apollo/client';
import { createBrowserHistory } from 'history';

import configureApolloClient from 'configureApolloClient';

import AgingImages from './AgingImages';

const range0 = '30';
const range1 = '90';
const range2 = '180';
const range3 = '365';

const result0 = 8;
const result1 = 1;
const result2 = 13;
const result3 = 18;

const mock = {
    data: {
        timeRange0: result0,
        timeRange1: result1,
        timeRange2: result2,
        timeRange3: result3,
    },
};

function setup() {
    const history = createBrowserHistory();

    cy.intercept('POST', '/api/graphql?opname=agingImagesQuery', (req) => req.reply(mock));

    cy.mount(
        <Router history={history}>
            <ApolloProvider client={configureApolloClient()}>
                <AgingImages />
            </ApolloProvider>
        </Router>
    );
}

describe(Cypress.spec.relative, () => {
    it('should render the correct number of images with default settings', () => {
        setup();

        // When all items are selected, the total should be equal to the total of all buckets
        // returned by the server
        cy.findByText(`${result0 + result1 + result2 + result3} Aging images`);

        // Each bar should display text that is specific to that time bucket, not
        // cumulative.
        cy.findByText(result0);
        cy.findByText(result1);
        cy.findByText(result2);
        cy.findByText(result3);
    });

    it('should render graph bars with the correct image counts when time buckets are toggled', () => {
        setup();
        cy.findByText(`${result0 + result1 + result2 + result3} Aging images`);

        cy.findByLabelText('Options').click();
        cy.findAllByLabelText('Toggle image time range').should('have.length', 4);

        // Disable the first bucket
        cy.findAllByLabelText('Toggle image time range').eq(0).click();

        // With the first item deselected, aging images < 90 days should no longer be present
        // in the chart or the card header
        cy.findByText(`${result1 + result2 + result3} Aging images`);

        // Test values at top of each bar
        cy.findByText(result0).should('not.exist');
        cy.findByText(result1);
        cy.findByText(result2);
        cy.findByText(result3);

        // Test display of x-axis, the first bucket should no longer be present
        cy.findByText(`${range1}-${range2} days`);
        cy.findByText(`${range2}-${range3} days`);
        cy.findByText(`>1 year`);

        // Re-enable the first bucket
        cy.findAllByLabelText('Toggle image time range').eq(0).click();
        // Disable the third bucket
        cy.findAllByLabelText('Toggle image time range').eq(2).click();

        // With the first item re-selected (regardless of the other selected items), the heading total
        // should revert to the original value.
        cy.findByText(`${result0 + result1 + result2 + result3} Aging images`);

        cy.findByText(result0);
        // The second bar in the chart should now contain values from the second and third buckets
        cy.findByText(result1 + result2);
        cy.findByText(result2).should('not.exist');
        cy.findByText(result3);

        // Test display of x-axis
        cy.findByText(`${range0}-${range1} days`);
        cy.findByText(`${range1}-${range3} days`);
        cy.findByText(`>1 year`);
    });

    it('links users to the correct filtered image list', () => {
        setup();

        cy.findByText(`${result0 + result1 + result2 + result3} Aging images`);

        // Check default links
        cy.findByText(`${range0}-${range1} days`).click();
        cy.url({ decode: true }).should('include', 's[Image Created Time]=30d-90d');

        cy.findByText(`${range1}-${range2} days`).click();
        cy.url({ decode: true }).should('include', 's[Image Created Time]=90d-180d');

        cy.findByText('>1 year').click();
        cy.url({ decode: true }).should('include', 's[Image Created Time]=>365d');

        // Deselect the second time range, merging the first and second time buckets
        cy.findByLabelText('Options').click();
        cy.findAllByLabelText('Toggle image time range').eq(1).click();
        cy.findByLabelText('Options').click();

        cy.findByText(`${range0}-${range2} days`).click();
        cy.url({ decode: true }).should('include', 's[Image Created Time]=30d-180d');
    });

    it('should contain a button that resets the widget options to default', () => {
        setup();

        cy.findByLabelText('Options').click();

        // All ranges enabled by default
        cy.findAllByLabelText('Toggle image time range').should('be.checked');

        function expectInputValues(range0, range1, range2, range3) {
            cy.findAllByLabelText('Image age in days').eq(0).should('have.value', range0);
            cy.findAllByLabelText('Image age in days').eq(1).should('have.value', range1);
            cy.findAllByLabelText('Image age in days').eq(2).should('have.value', range2);
            cy.findAllByLabelText('Image age in days').eq(3).should('have.value', range3);
        }

        // Defaults
        expectInputValues('30', '90', '180', '365');

        cy.findAllByLabelText('Toggle image time range').eq(0).click();
        cy.findAllByLabelText('Toggle image time range').eq(1).click();
        cy.findAllByLabelText('Image age in days').eq(1).type('{selectall}100');
        cy.findAllByLabelText('Image age in days').eq(2).type('{selectall}200');

        cy.findAllByLabelText('Toggle image time range').eq(0).should('not.be.checked');
        cy.findAllByLabelText('Toggle image time range').eq(1).should('not.be.checked');
        cy.findAllByLabelText('Toggle image time range').eq(2).should('be.checked');
        cy.findAllByLabelText('Toggle image time range').eq(3).should('be.checked');
        expectInputValues('30', '100', '200', '365');

        cy.findByLabelText('Revert to default options').click();
        cy.findByLabelText('Options').click();

        cy.findAllByLabelText('Toggle image time range').should('be.checked');
        expectInputValues('30', '90', '180', '365');
    });
});

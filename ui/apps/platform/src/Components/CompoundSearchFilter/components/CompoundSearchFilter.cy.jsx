import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import CompoundSearchFilter from './CompoundSearchFilter';
import {
    clusterSearchFilterConfig,
    deploymentSearchFilterConfig,
    imageComponentSearchFilterConfig,
    imageSearchFilterConfig,
    nodeComponentSearchFilterConfig,
} from '../types';

const selectors = {
    entitySelectToggle: 'button[aria-label="compound search filter entity selector toggle"]',
    entitySelectItems: 'div[aria-label="compound search filter entity selector menu"] ul li',
    entitySelectItem: (text) => `${selectors.entitySelectItems} button:contains(${text})`,
    attributeSelectToggle: 'button[aria-label="compound search filter attribute selector toggle"]',
    attributeSelectItems: 'div[aria-label="compound search filter attribute selector menu"] ul li',
    attributeSelectItem: (text) => `${selectors.attributeSelectItems} button:contains(${text})`,
};

const imageNameResponseMock = {
    data: {
        searchAutocomplete: [
            'docker.io/library/centos:7',
            'docker.io/library/centos:8',
            'quay.io/centos:7',
        ],
    },
};

function Wrapper({ config }) {
    return (
        <div className="pf-v5-u-p-md">
            <CompoundSearchFilter config={config} />
        </div>
    );
}

function setup(config) {
    cy.mount(
        <ComponentTestProviders>
            <Wrapper config={config} />
        </ComponentTestProviders>
    );
}

function mockAutocompleteResponse() {
    cy.intercept('POST', graphqlUrl('autocomplete'), (req) => {
        const query = req?.body?.variables?.query || '';
        const filterValue = query.includes(':') ? query.split(':')[1].replace('r/', '') : '';

        const response = {
            data: {
                searchAutocomplete: filterValue
                    ? imageNameResponseMock.data.searchAutocomplete.filter((value) =>
                          value.includes(filterValue)
                      )
                    : imageNameResponseMock.data.searchAutocomplete,
            },
        };

        req.reply(response);
    }).as('autocomplete');
}

describe(Cypress.spec.relative, () => {
    it('should display nothing in the entity selector', () => {
        const config = {};

        setup(config);

        cy.get(selectors.entitySelectToggle).should('not.exist');
    });

    it('should display the Image entity in the entity selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.entitySelectToggle).should('contain.text', 'Image');

        cy.get(selectors.entitySelectToggle).click();

        cy.get(selectors.entitySelectItems).should('have.length', 1);
        cy.get(selectors.entitySelectItems).eq(0).should('have.text', 'Image');
    });

    it('should display Image and Deployment entities in the entity selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
            Deployment: deploymentSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.entitySelectToggle).should('contain.text', 'Image');

        cy.get(selectors.entitySelectToggle).click();

        cy.get(selectors.entitySelectItems).should('have.length', 2);
        cy.get(selectors.entitySelectItems).eq(0).should('have.text', 'Image');
        cy.get(selectors.entitySelectItems).eq(1).should('have.text', 'Deployment');
    });

    it('should display Image, Deployment, and Cluster entities in the entity selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
            Deployment: deploymentSearchFilterConfig,
            Cluster: clusterSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.entitySelectToggle).should('contain.text', 'Image');

        cy.get(selectors.entitySelectToggle).click();

        cy.get(selectors.entitySelectItems).should('have.length', 3);
        cy.get(selectors.entitySelectItems).eq(0).should('have.text', 'Image');
        cy.get(selectors.entitySelectItems).eq(1).should('have.text', 'Deployment');
        cy.get(selectors.entitySelectItems).eq(2).should('have.text', 'Cluster');
    });

    it('should display Image attributes in the attribute selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
            Deployment: deploymentSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();

        cy.get(selectors.attributeSelectItems).should('have.length', 8);
        cy.get(selectors.attributeSelectItems).eq(0).should('have.text', 'Name');
        cy.get(selectors.attributeSelectItems).eq(1).should('have.text', 'Operating System');
        cy.get(selectors.attributeSelectItems).eq(2).should('have.text', 'Tag');
        cy.get(selectors.attributeSelectItems).eq(3).should('have.text', 'CVSS');
        cy.get(selectors.attributeSelectItems).eq(4).should('have.text', 'Label');
        cy.get(selectors.attributeSelectItems).eq(5).should('have.text', 'Created Time');
        cy.get(selectors.attributeSelectItems).eq(6).should('have.text', 'Scan Time');
        cy.get(selectors.attributeSelectItems).eq(7).should('have.text', 'Registry');
    });

    it('should display Deployment attributes in the attribute selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
            Deployment: deploymentSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.entitySelectToggle).click();
        cy.get(selectors.entitySelectItem('Deployment')).click();

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();

        cy.get(selectors.attributeSelectItems).should('have.length', 3);
        cy.get(selectors.attributeSelectItems).eq(0).should('have.text', 'Name');
        cy.get(selectors.attributeSelectItems).eq(1).should('have.text', 'Label');
        cy.get(selectors.attributeSelectItems).eq(2).should('have.text', 'Annotation');
    });

    it('should display the text input for the image tag search filter', () => {
        const config = {
            Image: imageSearchFilterConfig,
            NodeComponent: nodeComponentSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();
        cy.get(selectors.attributeSelectItem('Tag')).click();

        cy.get('input[aria-label="Filter results by image tag"]').should('exist');
    });

    it('should display the select input for the image component source search filter', () => {
        const config = {
            Image: imageSearchFilterConfig,
            ImageComponent: imageComponentSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.entitySelectToggle).click();
        cy.get(selectors.entitySelectItem('Image Component')).click();

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();
        cy.get(selectors.attributeSelectItem('Source')).click();

        cy.get('button[aria-label="Filter by source"]').click();

        const nodeComponenSourceSelectItems =
            'div[aria-label="Filter by source select menu"] ul li';

        cy.get(nodeComponenSourceSelectItems).should('have.length', 7);
        cy.get(nodeComponenSourceSelectItems).eq(0).should('have.text', 'OS');
        cy.get(nodeComponenSourceSelectItems).eq(1).should('have.text', 'Python');
        cy.get(nodeComponenSourceSelectItems).eq(2).should('have.text', 'Java');
        cy.get(nodeComponenSourceSelectItems).eq(3).should('have.text', 'Ruby');
        cy.get(nodeComponenSourceSelectItems).eq(4).should('have.text', 'Node js');
        cy.get(nodeComponenSourceSelectItems).eq(5).should('have.text', 'Dotnet Core Runtime');
        cy.get(nodeComponenSourceSelectItems).eq(6).should('have.text', 'Infrastructure');
    });

    it('should display the date-picker input for the image create time search filter', () => {
        const config = {
            Image: imageSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.attributeSelectToggle).click();
        cy.get(selectors.attributeSelectItem('Created Time')).click();

        // The date-picker input should be present
        cy.get('input[aria-label="Filter by date"]').should('exist');

        // Click on the date-picker toggle
        cy.get('button[aria-label="Filter by date toggle"]').click();

        // Select a month
        cy.get('div.pf-v5-c-calendar-month__header-month button').click();
        cy.get('button.pf-v5-c-menu__item:contains("January")').click();

        // Select a year
        cy.get('input[aria-label="Select year"]').clear();
        cy.get('input[aria-label="Select year"]').type('2034');

        // Select a day
        cy.get('button.pf-v5-c-calendar-month__date:contains("15")').click();

        // Check updated date value
        cy.get('input[aria-label="Filter by date"]').should('have.value', '2034-01-15');
    });

    it('should display the condition-number input for the image cvss filter', () => {
        const config = {
            Image: imageSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.attributeSelectToggle).click();
        cy.get(selectors.attributeSelectItem('CVSS')).click();

        // should have default values
        cy.get('button[aria-label="Condition selector toggle"]').should(
            'have.text',
            'Is greater than'
        );
        cy.get('input[aria-label="Condition value input"]').should('have.value', '0');

        // change condition and number value
        cy.get('button[aria-label="Condition selector toggle"]').click();
        cy.get('div[aria-label="Condition selector menu"] li button:contains("Is less than")')
            .filter((_, element) => {
                // Get exact value
                // @TODO: Could be a custom command
                return Cypress.$(element).text().trim() === 'Is less than';
            })
            .click();
        cy.get('input[aria-label="Condition value input"]').clear();
        cy.get('input[aria-label="Condition value input"]').type(9.9);
        cy.get('input[aria-label="Condition value input"]').blur();

        // should have new values
        cy.get('button[aria-label="Condition selector toggle"]').should(
            'have.text',
            'Is less than'
        );
        cy.get('input[aria-label="Condition value input"]').should('have.value', '9.9');

        // should increment
        cy.get('button[aria-label="Condition value plus button"]').click();
        cy.get('button[aria-label="Condition value plus button"]').should('be.disabled');
        cy.get('input[aria-label="Condition value input"]').should('have.value', '10');

        // should decrement
        cy.get('input[aria-label="Condition value input"]').clear();
        cy.get('input[aria-label="Condition value input"]').type(0.1);
        cy.get('button[aria-label="Condition value minus button"]').click();
        cy.get('button[aria-label="Condition value minus button"]').should('be.disabled');
        cy.get('input[aria-label="Condition value input"]').should('have.value', '0');
    });

    it('should display the autocomplete input for the image name filter', () => {
        mockAutocompleteResponse();

        const config = {
            Image: imageSearchFilterConfig,
        };

        const autocompleteMenuToggle =
            'div[aria-labelledby="Filter results menu toggle"] button[aria-label="Menu toggle"]';
        const autocompleteMenuItems = 'div[aria-label="Filter results select menu"] ul li';
        const autocompleteInput = 'input[aria-label="Filter results by image name"]';
        const autocompleteClearInputButton =
            'div[aria-labelledby="Filter results menu toggle"] button[aria-label="Clear input value"]';

        setup(config);

        cy.get(autocompleteMenuToggle).click();

        cy.wait('@autocomplete');

        cy.get(autocompleteMenuItems).should('have.length', 3);
        cy.get(autocompleteMenuItems).eq(0).should('have.text', 'docker.io/library/centos:7');
        cy.get(autocompleteMenuItems).eq(1).should('have.text', 'docker.io/library/centos:8');
        cy.get(autocompleteMenuItems).eq(2).should('have.text', 'quay.io/centos:7');

        cy.get(autocompleteMenuItems).eq(0).click();

        cy.get(autocompleteInput).should('have.value', 'docker.io/library/centos:7');

        cy.get(autocompleteClearInputButton).click();

        cy.get(autocompleteInput).should('have.value', '');

        cy.get(autocompleteInput).type('docker.io');

        cy.wait('@autocomplete');

        cy.get(autocompleteMenuItems).should('have.length', 2);
        cy.get(autocompleteMenuItems).eq(0).should('have.text', 'docker.io/library/centos:7');
        cy.get(autocompleteMenuItems).eq(1).should('have.text', 'docker.io/library/centos:8');
    });
});

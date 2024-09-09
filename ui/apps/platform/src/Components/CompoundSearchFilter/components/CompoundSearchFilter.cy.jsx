import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import CompoundSearchFilter from './CompoundSearchFilter';
import { nodeComponentAttributes } from '../attributes/nodeComponent';
import { imageAttributes } from '../attributes/image';
import { imageCVEAttributes } from '../attributes/imageCVE';
import { imageComponentAttributes } from '../attributes/imageComponent';
import { deploymentAttributes } from '../attributes/deployment';
import { clusterAttributes } from '../attributes/cluster';

const nodeComponentSearchFilterConfig = {
    displayName: 'Node component',
    searchCategory: 'NODE_COMPONENTS',
    attributes: nodeComponentAttributes,
};

const imageSearchFilterConfig = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: imageAttributes,
};

const imageCVESearchFilterConfig = {
    displayName: 'Image CVE',
    searchCategory: 'IMAGES_VULNERABILITIES',
    attributes: imageCVEAttributes,
};

const imageComponentSearchFilterConfig = {
    displayName: 'Image component',
    searchCategory: 'IMAGE_COMPONENTS',
    attributes: imageComponentAttributes,
};

const deploymentSearchFilterConfig = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: deploymentAttributes,
};

const clusterSearchFilterConfig = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: clusterAttributes,
};

const selectors = {
    entitySelectToggle: 'button[aria-label="compound search filter entity selector toggle"]',
    entitySelectItems: '[aria-label="compound search filter entity selector menu"] li',
    entitySelectItem: (text) => `${selectors.entitySelectItems} button:contains(${text})`,
    attributeSelectToggle: 'button[aria-label="compound search filter attribute selector toggle"]',
    attributeSelectItems: '[aria-label="compound search filter attribute selector menu"] li',
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

function Wrapper({ config, searchFilter, onSearch }) {
    return (
        <div className="pf-v5-u-p-md">
            <CompoundSearchFilter config={config} searchFilter={searchFilter} onSearch={onSearch} />
        </div>
    );
}

function setup(config, searchFilter, onSearch) {
    cy.mount(
        <ComponentTestProviders>
            <Wrapper config={config} searchFilter={searchFilter} onSearch={onSearch} />
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

/**
 * Selects a date in a date-picker component.
 *
 * @param {string} month - The full name of the month to select (e.g., "January", "February").
 * @param {string} day - The day of the month to select (e.g., "01", "15").
 * @param {string} year - The four-digit year to select (e.g., "2023").
 *
 */
function selectDatePickerDate(month, day, year) {
    // The date-picker input should be present
    cy.get('input[aria-label="Filter by date"]').should('exist');

    // Click on the date-picker toggle
    cy.get('button[aria-label="Filter by date toggle"]').click();

    // Select a month
    cy.get('div.pf-v5-c-calendar-month__header-month button').click();
    cy.get(`button.pf-v5-c-menu__item:contains("${month}")`).click();

    // Select a year
    cy.get('input[aria-label="Select year"]').clear();
    cy.get('input[aria-label="Select year"]').type(year);

    // Select a day
    cy.get(`button.pf-v5-c-calendar-month__date:contains("${day}")`).click();
}

describe(Cypress.spec.relative, () => {
    it('should display nothing in the entity selector', () => {
        const config = [];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.entitySelectToggle).should('not.exist');
    });

    it('should display Image and Deployment entities in the entity selector', () => {
        const config = [imageSearchFilterConfig, deploymentSearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.entitySelectToggle).should('contain.text', 'Image');

        cy.get(selectors.entitySelectToggle).click();

        cy.get(selectors.entitySelectItems).should('have.length', 2);
        cy.get(selectors.entitySelectItems).eq(0).should('have.text', 'Image');
        cy.get(selectors.entitySelectItems).eq(1).should('have.text', 'Deployment');
    });

    it('should display Image, Deployment, and Cluster entities in the entity selector', () => {
        const config = [
            imageSearchFilterConfig,
            deploymentSearchFilterConfig,
            clusterSearchFilterConfig,
        ];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.entitySelectToggle).should('contain.text', 'Image');

        cy.get(selectors.entitySelectToggle).click();

        cy.get(selectors.entitySelectItems).should('have.length', 3);
        cy.get(selectors.entitySelectItems).eq(0).should('have.text', 'Image');
        cy.get(selectors.entitySelectItems).eq(1).should('have.text', 'Deployment');
        cy.get(selectors.entitySelectItems).eq(2).should('have.text', 'Cluster');
    });

    it('should display nothing in the attributes selector', () => {
        const config = [
            {
                displayName: 'Image',
                searchCategory: 'IMAGES',
                attributes: [],
            },
        ];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.attributeSelectToggle).should('not.exist');
    });

    it('should display Image attributes in the attribute selector', () => {
        const config = [imageSearchFilterConfig, deploymentSearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();

        cy.get(selectors.attributeSelectItems).should('have.length', 5);
        cy.get(selectors.attributeSelectItems).eq(0).should('have.text', 'Name');
        cy.get(selectors.attributeSelectItems).eq(1).should('have.text', 'Operating system');
        cy.get(selectors.attributeSelectItems).eq(2).should('have.text', 'Tag');
        cy.get(selectors.attributeSelectItems).eq(3).should('have.text', 'Label');
        cy.get(selectors.attributeSelectItems).eq(4).should('have.text', 'Registry');
    });

    it('should display Deployment attributes in the attribute selector', () => {
        const config = [imageSearchFilterConfig, deploymentSearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.entitySelectToggle).click();
        cy.get(selectors.entitySelectItem('Deployment')).click();

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'ID');

        cy.get(selectors.attributeSelectToggle).click();

        cy.get(selectors.attributeSelectItems).should('have.length', 5);
        cy.get(selectors.attributeSelectItems).eq(0).should('have.text', 'ID');
        cy.get(selectors.attributeSelectItems).eq(1).should('have.text', 'Name');
        cy.get(selectors.attributeSelectItems).eq(2).should('have.text', 'Label');
        cy.get(selectors.attributeSelectItems).eq(3).should('have.text', 'Annotation');
        cy.get(selectors.attributeSelectItems).eq(4).should('have.text', 'Status');
    });

    it('should display the text input and correctly search for image tags', () => {
        const config = [imageSearchFilterConfig, nodeComponentSearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();
        cy.get(selectors.attributeSelectItem('Tag')).click();

        cy.get('input[aria-label="Filter results by Image tag"]').should('exist');

        cy.get('input[aria-label="Filter results by Image tag"]').clear();
        cy.get('input[aria-label="Filter results by Image tag"]').type('Tag 123');
        cy.get('button[aria-label="Apply text input to search"]').click();

        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'Image Tag',
            value: 'Tag 123',
        });
    });

    it('should display the select input and correctly search for image component source', () => {
        const config = [imageSearchFilterConfig, imageComponentSearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.entitySelectToggle).click();
        cy.get(selectors.entitySelectItem('Image component')).click();

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();
        cy.get(selectors.attributeSelectItem('Source')).click();

        cy.get('button[aria-label="Filter by Source"]').click();

        const imageComponenSourceSelectItems =
            'div[aria-label="Filter by Source select menu"] ul li';

        cy.get(imageComponenSourceSelectItems).should('have.length', 8);
        cy.get(imageComponenSourceSelectItems).eq(0).should('have.text', 'OS');
        cy.get(imageComponenSourceSelectItems).eq(1).should('have.text', 'Python');
        cy.get(imageComponenSourceSelectItems).eq(2).should('have.text', 'Java');
        cy.get(imageComponenSourceSelectItems).eq(3).should('have.text', 'Ruby');
        cy.get(imageComponenSourceSelectItems).eq(4).should('have.text', 'Node js');
        cy.get(imageComponenSourceSelectItems).eq(5).should('have.text', 'Go');
        cy.get(imageComponenSourceSelectItems).eq(6).should('have.text', 'Dotnet Core Runtime');
        cy.get(imageComponenSourceSelectItems).eq(7).should('have.text', 'Infrastructure');

        cy.get(imageComponenSourceSelectItems).eq(1).click();
        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'Component Source',
            value: 'PYTHON',
        });

        cy.get(imageComponenSourceSelectItems).eq(4).click();
        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'Component Source',
            value: 'NODEJS',
        });
    });

    it('should display the date-picker input and correctly search for image cve discovered time', () => {
        const config = [imageCVESearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.entitySelectToggle).click();
        cy.get(selectors.entitySelectItem('CVE')).click();

        cy.get(selectors.attributeSelectToggle).should('contain.text', 'Name');

        cy.get(selectors.attributeSelectToggle).click();
        cy.get(selectors.attributeSelectItem('Discovered time')).click();

        cy.get('button[aria-label="Condition selector toggle"]').should('have.text', 'On');

        cy.get('button[aria-label="Condition selector toggle"]').click();
        cy.get('[aria-label="Condition selector menu"] li button:contains("After")')
            .filter((_, element) => {
                // Get exact value
                // @TODO: Could be a custom command
                return Cypress.$(element).text().trim() === 'After';
            })
            .click();

        selectDatePickerDate('January', '15', '2034');

        cy.get('button[aria-label="Apply condition and date input to search"]').click();

        // Check updated date value
        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'CVE Created Time',
            value: '>01/15/2034',
        });

        cy.get('input[aria-label="Filter by date"]').should('have.value', '');
    });

    it('should display the condition-number input and correctly search for image cvss', () => {
        const config = [imageCVESearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        setup(config, searchFilter, onSearch);

        cy.get(selectors.entitySelectToggle).click();
        cy.get(selectors.entitySelectItem('CVE')).click();

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
        cy.get('[aria-label="Condition selector menu"] li button:contains("Is less than")')
            .filter((_, element) => {
                // Get exact value
                // @TODO: Could be a custom command
                return Cypress.$(element).text().trim() === 'Is less than';
            })
            .click();
        cy.get('input[aria-label="Condition value input"]').clear();
        cy.get('input[aria-label="Condition value input"]').type(9.9);
        cy.get('input[aria-label="Condition value input"]').blur();

        cy.get('button[aria-label="Apply condition and number input to search"]').click();
        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'CVSS',
            value: '<9.9',
        });

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

        cy.get('button[aria-label="Apply condition and number input to search"]').click();
        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'CVSS',
            value: '<10',
        });

        // should decrement
        cy.get('input[aria-label="Condition value input"]').clear();
        cy.get('input[aria-label="Condition value input"]').type(0.1);
        cy.get('button[aria-label="Condition value minus button"]').click();
        cy.get('button[aria-label="Condition value minus button"]').should('be.disabled');
        cy.get('input[aria-label="Condition value input"]').should('have.value', '0');

        cy.get('button[aria-label="Apply condition and number input to search"]').click();
        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'CVSS',
            value: '<0',
        });
    });

    it('should display the autocomplete input and correctly search for image name', () => {
        mockAutocompleteResponse();

        const config = [imageSearchFilterConfig];
        const onSearch = cy.stub().as('onSearch');
        const searchFilter = {};

        const autocompleteMenuToggle =
            'div[aria-labelledby="Filter results menu toggle"] button[aria-label="Menu toggle"]';
        const autocompleteMenuItems = '[aria-label="Filter results select menu"] li';
        const autocompleteInput = 'input[aria-label="Filter results by Image name"]';
        const autocompleteSearchButton = 'button[aria-label="Apply autocomplete input to search"]';

        setup(config, searchFilter, onSearch);

        cy.get(autocompleteMenuToggle).click();

        cy.wait('@autocomplete');

        cy.get(autocompleteMenuItems).should('have.length', 3);
        cy.get(autocompleteMenuItems).eq(0).should('have.text', 'docker.io/library/centos:7');
        cy.get(autocompleteMenuItems).eq(1).should('have.text', 'docker.io/library/centos:8');
        cy.get(autocompleteMenuItems).eq(2).should('have.text', 'quay.io/centos:7');

        cy.get(autocompleteMenuItems).eq(0).click();

        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'Image',
            value: 'docker.io/library/centos:7',
        });

        cy.get(autocompleteInput).should('have.value', '');

        cy.get(autocompleteInput).type('docker.io');

        cy.wait('@autocomplete');

        cy.get(autocompleteMenuItems).should('have.length', 2);
        cy.get(autocompleteMenuItems).eq(0).should('have.text', 'docker.io/library/centos:7');
        cy.get(autocompleteMenuItems).eq(1).should('have.text', 'docker.io/library/centos:8');

        cy.get(autocompleteSearchButton).click();
        cy.get('@onSearch').should('have.been.calledWithExactly', {
            action: 'ADD',
            category: 'Image',
            value: 'docker.io',
        });
    });
});

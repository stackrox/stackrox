import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';

import CompoundSearchFilter from './CompoundSearchFilter';
import {
    clusterSearchFilterConfig,
    deploymentSearchFilterConfig,
    imageSearchFilterConfig,
} from '../types';

const selectors = {
    entitySelectToggle: 'button[aria-label="compound search filter entity selector toggle"]',
    entitySelectItems: 'div[aria-label="compound search filter entity selector menu"] ul li',
    entitySelectItem: (text) => `${selectors.entitySelectItems} button:contains(${text})`,
    attributeSelectToggle: 'button[aria-label="compound search filter attribute selector toggle"]',
    attributeSelectItems: 'div[aria-label="compound search filter attribute selector menu"] ul li',
};

function Wrapper({ config }) {
    return <CompoundSearchFilter config={config} />;
}

function setup(config) {
    cy.mount(
        <ComponentTestProviders>
            <Wrapper config={config} />
        </ComponentTestProviders>
    );
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
});

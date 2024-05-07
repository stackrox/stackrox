import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';

import CompoundSearchFilter from './CompoundSearchFilter';
import {
    clusterSearchFilterConfig,
    deploymentSearchFilterConfig,
    imageSearchFilterConfig,
} from '../types';

const selectors = {
    selectToggle: 'button[aria-label="compound search filter entity selector toggle"]',
    selectItems: 'ul[aria-label="compound search filter entity selector items"] li',
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

        cy.get(selectors.selectToggle).should('not.exist');
    });

    it('should display the Image entity in the entity selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.selectToggle).should('contain.text', 'Image');

        cy.get(selectors.selectToggle).click();

        cy.get(selectors.selectItems).should('have.length', 1);
        cy.get(selectors.selectItems).eq(0).should('have.text', 'Image');
    });

    it('should display Image and Deployment entities in the entity selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
            Deployment: deploymentSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.selectToggle).should('contain.text', 'Image');

        cy.get(selectors.selectToggle).click();

        cy.get(selectors.selectItems).should('have.length', 2);
        cy.get(selectors.selectItems).eq(0).should('have.text', 'Image');
        cy.get(selectors.selectItems).eq(1).should('have.text', 'Deployment');
    });

    it('should display Image, Deployment, and Cluster entities in the entity selector', () => {
        const config = {
            Image: imageSearchFilterConfig,
            Deployment: deploymentSearchFilterConfig,
            Cluster: clusterSearchFilterConfig,
        };

        setup(config);

        cy.get(selectors.selectToggle).should('contain.text', 'Image');

        cy.get(selectors.selectToggle).click();

        cy.get(selectors.selectItems).should('have.length', 3);
        cy.get(selectors.selectItems).eq(0).should('have.text', 'Image');
        cy.get(selectors.selectItems).eq(1).should('have.text', 'Deployment');
        cy.get(selectors.selectItems).eq(2).should('have.text', 'Cluster');
    });
});

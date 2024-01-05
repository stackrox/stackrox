import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';
import { graphqlUrl } from 'test-utils/apiEndpoints';
import { vulnManagementImagesPath, vulnManagementPath } from 'routePaths';

import ImagesAtMostRisk from './ImagesAtMostRisk';

/**
 * @typedef {typeof vulnCounts} VulnCounts
 */

/**
 * @param {string} id
 * @param {string} remote
 * @param {string} fullName
 * @param {number} priority
 * @param {VulnCounts} imageVulnerabilityCounter
 */
function makeMockImage(id, remote, fullName, priority, imageVulnerabilityCounter) {
    return {
        id,
        name: { remote, fullName },
        priority,
        imageVulnerabilityCounter,
    };
}

const totalImportant = 120;
const fixableImportant = 80;
const totalCritical = 100;
const fixableCritical = 60;

const vulnCounts = {
    important: {
        total: totalImportant,
        fixable: fixableImportant,
    },
    critical: {
        total: totalCritical,
        fixable: fixableCritical,
    },
};

const mockImages = [1, 2, 3, 4, 5, 6].map((n) =>
    makeMockImage(`${n}`, `name-${n}`, `reg/name-${n}:tag`, n, vulnCounts)
);

const mock = {
    data: {
        images: mockImages,
    },
};

function setup() {
    cy.intercept('POST', graphqlUrl('getImagesAtMostRisk'), (req) => {
        req.reply(mock);
    });

    cy.mount(
        <ComponentTestProviders>
            <ImagesAtMostRisk />
        </ComponentTestProviders>
    );
}

describe(Cypress.spec.relative, () => {
    it('should render the correct title based on selected options', () => {
        setup();

        // Default is display all images
        cy.findByText('Images at most risk');

        // Change to display only active images
        cy.findByLabelText('Options').click();
        cy.findByText('Active images').click();

        cy.findByText('Active images at most risk');
    });

    it('should render the correct text and number of CVEs under each column', () => {
        setup();

        // Note that in this case the mock data uses the same number of CVEs for every image
        // so we will expect multiple elements matching the below text queries

        // Default should show fixable CVEs
        cy.findAllByText(`${fixableCritical} fixable`).should('have.length', mockImages.length);
        cy.findAllByText(`${fixableImportant} fixable`).should('have.length', mockImages.length);

        // Switch to show total CVEs
        cy.findByLabelText('Options').click();
        cy.findByText('All CVEs').click();

        cy.findAllByText(`${totalCritical} CVEs`).should('have.length', mockImages.length);
        cy.findAllByText(`${totalImportant} CVEs`).should('have.length', mockImages.length);
    });

    it('should link to the appropriate pages in VulnMgmt', () => {
        setup();

        cy.findByText('Images at most risk');

        // Click on the link matching the second image
        const secondImageInList = mockImages.at(1);
        cy.findByText(secondImageInList.name.remote).click();

        cy.location('pathname').should('eq', `${vulnManagementPath}/image/${secondImageInList.id}`);
        cy.location('hash').should('eq', '#image-findings');

        cy.findByText('View all').click();
        cy.location('pathname').should('eq', `${vulnManagementImagesPath}`);
    });

    it('should contain a button that resets the widget options to default', () => {
        setup();

        const getMenuButton = (name) => cy.findByRole('button', { name });

        cy.findByLabelText('Options').click();

        getMenuButton('Fixable CVEs').should('have.attr', 'aria-pressed', 'true');
        getMenuButton('All CVEs').should('have.attr', 'aria-pressed', 'false');
        getMenuButton('Active images').should('have.attr', 'aria-pressed', 'false');
        getMenuButton('All images').should('have.attr', 'aria-pressed', 'true');

        // Change some options
        getMenuButton('All CVEs').click();
        getMenuButton('Active images').click();

        getMenuButton('Fixable CVEs').should('have.attr', 'aria-pressed', 'false');
        getMenuButton('All CVEs').should('have.attr', 'aria-pressed', 'true');
        getMenuButton('Active images').should('have.attr', 'aria-pressed', 'true');
        getMenuButton('All images').should('have.attr', 'aria-pressed', 'false');

        cy.findByLabelText('Revert to default options').click();

        // re-open menu
        cy.findByLabelText('Options').click();

        // Check return to defaults
        getMenuButton('Fixable CVEs').should('have.attr', 'aria-pressed', 'true');
        getMenuButton('All CVEs').should('have.attr', 'aria-pressed', 'false');
        getMenuButton('Active images').should('have.attr', 'aria-pressed', 'false');
        getMenuButton('All images').should('have.attr', 'aria-pressed', 'true');
    });
});

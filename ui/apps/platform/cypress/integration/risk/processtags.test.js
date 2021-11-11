import randomstring from 'randomstring';

import { selectors, url } from '../../constants/RiskPage';

import { selectors as searchSelectors } from '../../constants/SearchPage';
import search from '../../selectors/search';
import * as api from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';

function setRoutes() {
    cy.intercept('GET', api.risks.riskyDeployments).as('deployments');
    cy.intercept('GET', api.risks.fetchDeploymentWithRisk).as('getDeployment');
    cy.intercept('POST', api.graphql(api.risks.graphqlOps.getProcessTags)).as('getTags');
    cy.intercept('POST', api.graphql(api.risks.graphqlOps.autocomplete)).as('tagsAutocomplete');
}

function openDeployment(deploymentName) {
    cy.visit(url);
    cy.wait('@deployments');

    cy.get(`${selectors.table.rows}:contains(${deploymentName})`).click();
    cy.wait('@getDeployment');
}

function unfoldFirstProcessCard() {
    cy.get(selectors.sidePanel.processDiscoveryTab).click();
    cy.get(selectors.sidePanel.firstProcessCard.header).click();
    cy.wait(['@getTags', '@tagsAutocomplete']);
}

function addTagToTheFirstProcessInDeployment(deploymentName, tag) {
    openDeployment(deploymentName);
    unfoldFirstProcessCard();

    cy.get(selectors.sidePanel.firstProcessCard.tags.input).type(`${tag}{enter}`);
    cy.wait(['@getTags', '@tagsAutocomplete']);
}

describe(
    'Risk Page Process Tags',
    {
        retries: {
            runMode: 1,
            openMode: 0,
        },
    },
    () => {
        withAuth();

        it('should add tag without allowing duplicates', () => {
            setRoutes();

            const tag = randomstring.generate(7);
            addTagToTheFirstProcessInDeployment('central', tag);
            // do it again to check that no duplicate tags can be added
            cy.get(selectors.sidePanel.firstProcessCard.tags.input).type(`${tag}{enter}`);

            // pressing {enter} won't save the tag, only one would be displayed as tag chip
            cy.get(selectors.sidePanel.firstProcessCard.tags.values)
                .contains(tag)
                .should('have.length', 1);
        });

        it('should search by a process tag', () => {
            setRoutes();

            const tag = randomstring.generate(7);
            addTagToTheFirstProcessInDeployment('central', tag);
            cy.get(selectors.sidePanel.cancelButton).click(); // close the side panel, we don't need it right now

            cy.get(searchSelectors.pageSearch.input).type('Process Tag{enter}');
            cy.get(searchSelectors.pageSearch.input).type(`${tag}{enter}`);
            cy.wait('@deployments');

            cy.get(selectors.table.rows).should('have.length', 1);
            cy.get(selectors.table.rows).contains('central');
        });

        // TODO: figure out the flake
        //       where it fails even though the passing condition is on the screen
        //       (started after upgrade to apollo-client 3.x)
        it.skip('should suggest autocompletion for existing tags', () => {
            setRoutes();

            const tag = randomstring.generate(7);
            addTagToTheFirstProcessInDeployment('central', tag);

            // select some other violation
            openDeployment('sensor');
            unfoldFirstProcessCard();

            cy.get(selectors.sidePanel.firstProcessCard.tags.input).type(`${tag.charAt(0)}`);
            cy.get(selectors.sidePanel.firstProcessCard.tags.options).contains(tag);
            cy.get(selectors.sidePanel.cancelButton).click(); // close the side panel, we don't need it right now

            cy.get(searchSelectors.pageSearch.input).type('Process Tag:{enter}');
            cy.get(searchSelectors.pageSearch.input).type(`${tag.charAt(0)}`);
            cy.get(search.options).contains(tag);
        });

        it('should remove tag', () => {
            setRoutes();

            const tag = randomstring.generate(7);
            addTagToTheFirstProcessInDeployment('central', tag);

            cy.get(selectors.sidePanel.firstProcessCard.tags.removeValueButton(tag)).click();
            cy.wait(['@getTags', '@tagsAutocomplete']);

            cy.get(`${selectors.sidePanel.firstProcessCard.tags.values}:contains("${tag}")`).should(
                'not.exist'
            );
        });
    }
);

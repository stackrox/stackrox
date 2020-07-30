import randomstring from 'randomstring';

import { selectors, url } from '../../constants/RiskPage';

import * as api from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';

const commentsSelectors = selectors.sidePanel.firstProcessCard.comments;

function setRoutes() {
    cy.server();
    cy.route('GET', api.risks.riskyDeployments).as('deployments');
    cy.route('GET', api.risks.getDeploymentWithRisk).as('getDeployment');
    cy.route('POST', api.graphql(api.risks.graphqlOps.getProcessComments)).as('getComments');
}

function openDeploymentFirstProcess(deploymentName) {
    cy.visit(url);
    cy.wait('@deployments');

    cy.get(`${selectors.table.rows}:contains(${deploymentName})`).click();
    cy.wait('@getDeployment');

    cy.get(selectors.sidePanel.processDiscoveryTab).click();
    cy.get(selectors.sidePanel.firstProcessCard.header).click();
    cy.wait('@getComments');
}

function deleteLastComment() {
    cy.get(commentsSelectors.lastComment.deleteButton).click();
    cy.get(selectors.commentsDialog.yesButton).click();
    cy.wait('@getComments');
}

describe('Risk Page Process Comments', () => {
    withAuth();

    it('should add new comment with a link and delete', () => {
        setRoutes();
        openDeploymentFirstProcess('central');

        const link = 'http://nowhere.org';
        const mark = randomstring.generate(7);
        const comment = `${link} ${mark} ${link} not a link ${link}`;
        cy.get(commentsSelectors.newButton).click();
        cy.get(commentsSelectors.newComment.textArea).type(comment);
        cy.get(commentsSelectors.newComment.saveButton).click();
        cy.wait('@getComments');

        cy.get(commentsSelectors.lastComment.userName).should('have.text', 'ui_tests');
        cy.get(commentsSelectors.lastComment.dateAndEditedStatus).should((date) => {
            const created = Cypress.moment(date.text(), 'MM/DD/YYYY | h:mm:ssA');
            const now = Cypress.moment();
            expect(now.diff(created, 'minutes')).to.equal(0); // let's hope server time is fine
        });
        cy.get(commentsSelectors.lastComment.message).should('have.text', comment);
        cy.get(commentsSelectors.lastComment.links).should('have.length', 3);
        cy.get(commentsSelectors.lastComment.links).each((a) => {
            expect(a).to.have.text(link);
            expect(a).to.have.attr('href', link);
        });

        deleteLastComment();

        cy.get(`${commentsSelectors.allComments}:contains("${mark}")`).should('not.exist');
    });

    it('should not allow saving empty comment', () => {
        setRoutes();
        openDeploymentFirstProcess('central');

        cy.get(commentsSelectors.newButton).click();
        cy.get(commentsSelectors.newComment.textArea).type('   ');
        cy.get(commentsSelectors.newComment.saveButton).click();

        cy.get(commentsSelectors.newComment.error).should('have.text', 'This field is required');
    });

    it('should edit existing comment', () => {
        setRoutes();
        openDeploymentFirstProcess('central');

        cy.get(commentsSelectors.newButton).click();
        cy.get(commentsSelectors.newComment.textArea).type('My comment');
        cy.get(commentsSelectors.newComment.saveButton).click();
        cy.wait('@getComments');

        cy.get(commentsSelectors.lastComment.editButton).click();
        cy.get(commentsSelectors.lastComment.textArea).type('{end} (updated)');
        cy.get(commentsSelectors.lastComment.saveButton).click();
        cy.wait('@getComments');

        cy.get(commentsSelectors.lastComment.message).should('have.text', 'My comment (updated)');
        cy.get(commentsSelectors.lastComment.dateAndEditedStatus).should(
            'contain.text',
            '(edited)'
        );

        deleteLastComment();
    });

    it('should not show edit and delete buttons if no permissions', () => {
        setRoutes();
        cy.route(
            'POST',
            api.graphql(api.risks.graphqlOps.getProcessComments),
            'fixture:risks/processComments.json'
        ).as('getComments');

        openDeploymentFirstProcess('central');

        cy.get(commentsSelectors.lastComment.message).should(
            'have.text',
            'Not editable / delete-able comment'
        );
        cy.get(commentsSelectors.lastComment.editButton).should('not.exist');
        cy.get(commentsSelectors.lastComment.deleteButton).should('not.exist');
    });
});

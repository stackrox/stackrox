import randomstring from 'randomstring';

import { selectors, url } from '../../constants/ViolationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

const commentsSelectors = selectors.details.comments;

function setRoutes() {
    cy.server();
    cy.route('GET', api.alerts.alerts).as('alerts');
    cy.route('GET', api.alerts.alertById).as('alertById');
    cy.route('POST', api.graphql(api.alerts.graphqlOps.getComments)).as('getComments');
}

function visitViolationsListPage() {
    cy.visit(url);
    cy.wait('@alerts');
}

function deleteLastComment() {
    cy.get(commentsSelectors.lastComment.deleteButton).click();
    cy.get(selectors.commentsDialog.deleteButton).click();
    cy.wait('@getComments');
}

describe(
    'Violation Page: Comments',
    {
        retries: {
            runMode: 1,
            openMode: 0,
        },
    },
    () => {
        withAuth();

        it('should add new comment with a link and delete', () => {
            setRoutes();
            visitViolationsListPage();

            cy.get(selectors.firstTableRowLink).then(($a) => {
                const href = $a.prop('href');

                cy.visit(href);
                cy.wait('@alertById');
                cy.wait('@getComments');

                const link = 'http://nowhere.org';
                const mark = randomstring.generate(7);
                const comment = `${link} ${mark} ${link} not a link ${link}`;
                cy.get(commentsSelectors.newButton).click();
                cy.get(commentsSelectors.newComment.textArea).type(comment);
                cy.get(commentsSelectors.newComment.saveButton).click();
                cy.wait('@getComments');

                cy.get(commentsSelectors.lastComment.userName).should('have.text', 'ui_tests');
                cy.get(commentsSelectors.lastComment.dateAndEditedStatus).should((date) => {
                    // Date matches 'MM/DD/YYYY | h:mm:ssA' format.
                    expect(/\d{2}\/\d{2}\/\d{4} | \d{1,2}:\d{2}:\d{2}(AM|PM)/.test(date.text())).to
                        .be.true;
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
        });

        it('should not allow saving empty comment', () => {
            setRoutes();
            visitViolationsListPage();

            cy.get(selectors.firstTableRowLink).then(($a) => {
                const href = $a.prop('href');

                cy.visit(href);
                cy.wait('@alertById');
                cy.wait('@getComments');

                cy.get(commentsSelectors.newButton).click();
                cy.get(commentsSelectors.newComment.textArea).type('   ');
                cy.get(commentsSelectors.newComment.saveButton).should('be.disabled');

                cy.get(commentsSelectors.newComment.error).should(
                    'have.text',
                    'This field is required'
                );
            });
        });

        it('should edit existing comment', () => {
            setRoutes();
            visitViolationsListPage();

            cy.get(selectors.firstTableRowLink).then(($a) => {
                const href = $a.prop('href');

                cy.log(href);
                cy.visit(href);
                cy.wait('@alertById');
                cy.wait('@getComments');

                cy.get(commentsSelectors.newButton).click();
                cy.get(commentsSelectors.newComment.textArea).type('My comment');
                cy.get(commentsSelectors.newComment.saveButton).click();
                cy.wait('@getComments');

                // first try to cancel changes
                cy.get(commentsSelectors.lastComment.editButton).click();
                cy.get(commentsSelectors.lastComment.textArea).type('{end} (updated)');
                cy.get(commentsSelectors.lastComment.cancelButton).click();

                cy.get(commentsSelectors.lastComment.message).should('have.text', 'My comment');
                cy.get(commentsSelectors.lastComment.dateAndEditedStatus).should(
                    'not.contain.text',
                    '(edited)'
                );

                // let do it with saving now
                cy.get(commentsSelectors.lastComment.editButton).click();
                cy.get(commentsSelectors.lastComment.textArea).type('{end} (updated)');
                cy.get(commentsSelectors.lastComment.saveButton).click();
                cy.wait('@getComments');

                cy.get(commentsSelectors.lastComment.message).should(
                    'have.text',
                    'My comment (updated)'
                );
                cy.get(commentsSelectors.lastComment.dateAndEditedStatus).should(
                    'contain.text',
                    '(edited)'
                );

                deleteLastComment();
            });
        });

        it('should not show edit and delete buttons if no permissions', () => {
            setRoutes();
            cy.route(
                'POST',
                api.graphql(api.alerts.graphqlOps.getComments),
                'fixture:alerts/comments.json'
            ).as('getComments');

            visitViolationsListPage();

            cy.get(selectors.firstTableRowLink).then(($a) => {
                const href = $a.prop('href');

                cy.visit(href);
                cy.wait('@alertById');
                cy.wait('@getComments');

                cy.get(commentsSelectors.lastComment.message).should(
                    'have.text',
                    'Not editable / delete-able comment'
                );
                cy.get(commentsSelectors.lastComment.editButton).should('not.exist');
                cy.get(commentsSelectors.lastComment.deleteButton).should('not.exist');
            });
        });
    }
);

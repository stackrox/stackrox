import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import AdministrationEventsPage from './AdministrationEventsPage';

const errorLevel = 'ADMINISTRATION_EVENT_LEVEL_ERROR';
const warningLevel = 'ADMINISTRATION_EVENT_LEVEL_WARNING';

function makeEvent(domain, level) {
    return {
        id: domain,
        domain,
        level,
        type: 'ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE',
        message: domain,
        hint: '',
        resource: { type: 'Image', id: '', name: 'example.com/image:latest' },
        numOccurrences: '1',
        createdAt: '2026-11-01T12:00:00Z',
        lastOccurredAt: '2026-11-01T12:00:00Z',
    };
}

function selectLevel(level) {
    cy.findByRole('button', { name: 'Level filter menu toggle' }).click();
    cy.findByRole('option', { name: level, exact: true }).click();
}

describe(Cypress.spec.relative, () => {
    for (const statusCode of [200, 500]) {
        it(`ignores an older filter response with status ${statusCode}`, () => {
            let releaseOldResponse;

            cy.intercept('POST', graphqlUrl('autocomplete'), { data: { searchAutocomplete: [] } });
            cy.intercept('GET', '/v1/count/administration/events?*', { count: 1 });
            cy.intercept('GET', '/v1/administration/events?*', (req) => {
                const level = req.query['filter.level'];
                if (level === errorLevel) {
                    req.alias = 'oldFilter';
                    return new Cypress.Promise((resolve) => {
                        releaseOldResponse = () => {
                            req.reply({
                                statusCode,
                                body:
                                    statusCode === 200
                                        ? { events: [makeEvent('Old result', errorLevel)] }
                                        : { message: 'Old request failed' },
                            });
                            resolve();
                        };
                    });
                }

                req.alias = level === warningLevel ? 'newFilter' : 'initialEvents';
                req.reply({ events: [makeEvent('Current result', warningLevel)] });
                return undefined;
            });

            window.history.pushState({}, document.title, '/main/administration-events');
            cy.mount(
                <ComponentTestProvider>
                    <AdministrationEventsPage />
                </ComponentTestProvider>
            );
            cy.wait('@initialEvents');
            selectLevel('Error');
            cy.wrap(null).should(() => {
                expect(releaseOldResponse).to.be.a('function');
            });
            selectLevel('Warning');
            cy.wait('@newFilter');
            cy.findByRole('link', { name: 'Current result' }).should('exist');

            cy.then(() => releaseOldResponse());
            cy.wait('@oldFilter');
            cy.findByRole('link', { name: 'Current result' }).should('exist');
            cy.findByRole('link', { name: 'Old result' }).should('not.exist');
            cy.findByText('Unable to fetch administration events').should('not.exist');
        });
    }
});

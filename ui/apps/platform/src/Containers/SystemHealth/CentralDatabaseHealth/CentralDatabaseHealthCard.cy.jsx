import React from 'react';

import CentralDatabaseHealthCard from './CentralDatabaseHealthCard';

function patchDatabaseStatusResponse(overrides = {}) {
    return cy.intercept('GET', '/v1/database/status', (req) => {
        req.reply({
            databaseAvailable: true,
            databaseType: 'PostgresDB',
            databaseVersion: '15.0',
            databaseIsExternal: false,
            ...overrides,
        });
    });
}

const cardHeader = '.pf-v5-c-card__header';
const cardBody = '.pf-v5-c-card__body';

describe(Cypress.spec.relative, () => {
    it('should not show a warning as the baseline', () => {
        patchDatabaseStatusResponse();
        cy.mount(<CentralDatabaseHealthCard />);
        cy.get(`${cardHeader}:contains("no errors")`);
        cy.get(`${cardHeader}:contains("PostgresDB 15.0")`);
    });

    it('should not show a warning if the database is external', () => {
        patchDatabaseStatusResponse({
            // Incorrect version
            databaseVersion: '13.14',
            // But external (ACSCS)
            databaseIsExternal: true,
        });

        cy.mount(<CentralDatabaseHealthCard />);
        cy.get(`${cardHeader}:contains("no errors")`);
        cy.get(`${cardHeader}:contains("PostgresDB 13.14")`);
    });

    it('should show a warning when a outdated version of the internal database is detected', () => {
        patchDatabaseStatusResponse({
            // Incorrect version
            databaseVersion: '13.14',
            // On an internal database
            databaseIsExternal: false,
        });

        cy.mount(<CentralDatabaseHealthCard />);
        cy.get(`${cardHeader}:contains("warning")`);
        cy.get(`${cardHeader}:contains("PostgresDB 13.14")`);
        cy.get(`${cardBody}:contains("Running an unsupported configuration of PostgreSQL")`);
    });
});

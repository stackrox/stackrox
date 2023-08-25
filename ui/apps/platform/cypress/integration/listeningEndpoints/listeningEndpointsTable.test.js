import withAuth from '../../helpers/basicAuth';
import { addEntityFilter, visitListeningEndpointsFromLeftNav } from './ListeningEndpoints.helpers';
import selectors from './ListeningEndpoints.selectors';

describe('Listening endpoints page table', () => {
    withAuth();

    it('should correctly display listening endpoint information for central-db', () => {
        visitListeningEndpointsFromLeftNav();

        addEntityFilter('Namespace', 'stackrox');

        addEntityFilter('Deployment', 'central-db');

        // assert that only one row is displayed in the table, each row is contained in its own tbody
        cy.get(`${selectors.deploymentTable} > tbody`).should('have.length', 1);

        const centralDbRowSelector = `${
            selectors.deploymentTable
        } > tbody:has(${selectors.tableRowWithValueForColumn('Deployment', 'central-db')})`;

        const centralDbProcessTableSelector = `${centralDbRowSelector} ${selectors.processTable}`;

        // Expand the row to show the process table
        cy.get(`${centralDbRowSelector} ${selectors.expandableRowToggle}`).click();

        // asset that there is only one row in the process table
        cy.get(`${centralDbProcessTableSelector} > tbody`).should('have.length', 1);
        // assert that the row contains a process name of 'postgres' and a port of '5432'
        cy.get(
            `${centralDbProcessTableSelector} ${selectors.tableRowWithValueForColumn(
                'Exec file path',
                '/usr/pgsql-13/bin/postgres'
            )}`
        );
        cy.get(
            `${centralDbProcessTableSelector} ${selectors.tableRowWithValueForColumn(
                'Port',
                '5432'
            )}`
        );
    });
});

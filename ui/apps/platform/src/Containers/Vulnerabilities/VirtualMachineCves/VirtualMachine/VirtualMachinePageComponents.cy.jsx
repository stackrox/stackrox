import { MemoryRouter } from 'react-router-dom-v5-compat';

import VirtualMachinePageComponents from './VirtualMachinePageComponents';

const virtualMachineId = '11111111-1111-4111-8111-111111111111';

describe(Cypress.spec.relative, () => {
    it('shows first-scan progress and fetches components when the scan completes', () => {
        let scanPending = true;
        cy.clock();
        cy.intercept('GET', `/v2/virtualmachines/${virtualMachineId}/components?*`, (req) => {
            req.reply({
                scanPending,
                totalCount: scanPending ? 0 : 1,
                components: scanPending
                    ? []
                    : [
                          {
                              id: 'component-1',
                              name: 'curl',
                              version: '8.0',
                              source: 'OS',
                              scanStatus: 'SCANNED',
                              cveCount: 0,
                          },
                      ],
            });
        }).as('getComponents');

        cy.mount(
            <MemoryRouter>
                <VirtualMachinePageComponents virtualMachineId={virtualMachineId} />
            </MemoryRouter>
        );
        cy.wait('@getComponents');
        cy.findByRole('heading', { name: 'Scan in progress' }).should('be.visible');
        cy.findByText('No components were detected for this virtual machine').should('not.exist');

        cy.then(() => {
            scanPending = false;
        });
        cy.tick(5000);
        cy.wait('@getComponents');
        cy.findByRole('cell', { name: 'curl' }).should('be.visible');
        cy.findByRole('heading', { name: 'Scan in progress' }).should('not.exist');
    });
});

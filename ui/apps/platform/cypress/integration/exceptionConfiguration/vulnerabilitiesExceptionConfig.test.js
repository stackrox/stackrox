import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    selectSingleCveForException,
    visitWorkloadCveOverview,
} from '../vulnerabilities/workloadCves/WorkloadCves.helpers';
import {
    resetExceptionConfig,
    visitExceptionConfig,
    visitExceptionConfigWithPermissions,
} from './ExceptionConfig.helpers';
import { vulnerabilitiesConfigSelectors as selectors } from './ExceptionConfig.selectors';

describe('Vulnerabilities Exception Configuration', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')) {
            this.skip();
        }
    });

    beforeEach(() => {
        if (hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')) {
            resetExceptionConfig();
        }
    });

    after(() => {
        if (hasFeatureFlag('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL')) {
            resetExceptionConfig();
        }
    });

    it('should correctly handle RBAC for the vulnerability exception config', () => {
        cy.fixture('auth/mypermissionsMinimalAccess.json').then(({ resourceToAccess }) => {
            // User with no access
            visitExceptionConfigWithPermissions('vulnerabilities', {
                ...resourceToAccess,
                Administration: 'NO_ACCESS',
            });

            cy.get('h1:contains("Cannot find the page")');

            // User with read-only access
            visitExceptionConfigWithPermissions('vulnerabilities', {
                ...resourceToAccess,
                Administration: 'READ_ACCESS',
            });

            cy.get(`h1:contains("Exception configuration")`);
            cy.get(selectors.saveButton).should('not.exist');

            for (let i = 0; i < 4; i += 1) {
                cy.get(selectors.dayOptionInput(i)).should('be.disabled');
                cy.get(selectors.dayOptionEnabledSwitch(i)).should('be.disabled');
            }

            cy.get(selectors.indefiniteOptionEnabledSwitch).should('be.disabled');
            cy.get(selectors.whenAllCveFixableSwitch).should('be.disabled');
            cy.get(selectors.whenAnyCveFixableSwitch).should('be.disabled');
            cy.get(selectors.customDateSwitch).should('be.disabled');
        });
    });

    it('should load the default config and allow modification', () => {
        visitExceptionConfig('vulnerabilities');

        cy.get(selectors.dayOptionEnabledSwitch(0)).check({ force: true });
        cy.get(selectors.dayOptionInput(0)).type('{selectall}20');
        cy.get(selectors.dayOptionEnabledSwitch(1)).check({ force: true });
        cy.get(selectors.dayOptionInput(1)).type('{selectall}40');
        cy.get(selectors.dayOptionEnabledSwitch(2)).check({ force: true });
        cy.get(selectors.dayOptionInput(2)).type('{selectall}60');

        cy.get(selectors.dayOptionEnabledSwitch(3)).uncheck({ force: true });
        cy.get(selectors.dayOptionInput(3)).should('be.disabled');

        cy.get(selectors.whenAllCveFixableSwitch).check({ force: true });
        cy.get(selectors.whenAnyCveFixableSwitch).uncheck({ force: true });
        cy.get(selectors.customDateSwitch).check({ force: true });
        cy.get(selectors.indefiniteOptionEnabledSwitch).check({ force: true });

        cy.get(selectors.saveButton).click();
        cy.get('.pf-c-alert:contains("The configuration was updated successfully")');

        // Refresh the page to make sure options are persisted
        visitExceptionConfig('vulnerabilities');

        cy.get(selectors.dayOptionInput(0)).should('have.value', '20');
        cy.get(selectors.dayOptionEnabledSwitch(0)).should('be.checked');
        cy.get(selectors.dayOptionInput(1)).should('have.value', '40');
        cy.get(selectors.dayOptionEnabledSwitch(1)).should('be.checked');
        cy.get(selectors.dayOptionInput(2)).should('have.value', '60');
        cy.get(selectors.dayOptionEnabledSwitch(2)).should('be.checked');

        cy.get(selectors.dayOptionInput(3)).should('not.be.checked');

        cy.get(selectors.whenAllCveFixableSwitch).should('be.checked');
        cy.get(selectors.whenAnyCveFixableSwitch).should('not.be.checked');
        cy.get(selectors.customDateSwitch).should('be.checked');
        cy.get(selectors.indefiniteOptionEnabledSwitch).should('be.checked');
    });

    it('should reflect an updated exception config in the Workload CVE exception flow', () => {
        // Apply global exception config options, enable half and disable the other half
        visitExceptionConfig('vulnerabilities');
        [0, 1, 2, 3].forEach((index) => {
            cy.get(selectors.dayOptionEnabledSwitch(index)).uncheck({ force: true });
        });
        cy.get(selectors.whenAllCveFixableSwitch).check({ force: true });
        cy.get(selectors.whenAnyCveFixableSwitch).check({ force: true });
        cy.get(selectors.customDateSwitch).check({ force: true });
        cy.get(selectors.indefiniteOptionEnabledSwitch).check({ force: true });

        cy.get(selectors.saveButton).click();
        cy.get('.pf-c-alert:contains("The configuration was updated successfully")');

        // Visit the Workload CVE page, open a deferral modal, and verify that the specified options are available
        visitWorkloadCveOverview();
        selectSingleCveForException('DEFERRAL');
        cy.get('button:contains("Options")').click();

        cy.get('label')
            .contains(/For \d+ days/)
            .should('have.length', 0);
        cy.get("label:contains('When any CVE is fixable')");
        cy.get("label:contains('When all CVEs are fixable')");
        cy.get("label:contains('Until a specific date')");
        cy.get("label:contains('Indefinitely')");

        // Revisit the config page and enable all day options, and disable the previously enabled options
        visitExceptionConfig('vulnerabilities');
        [0, 1, 2, 3].forEach((index) => {
            cy.get(selectors.dayOptionEnabledSwitch(index)).check({ force: true });
            cy.get(selectors.dayOptionInput(index)).type(`{selectall}${index + 1}`);
        });
        cy.get(selectors.whenAllCveFixableSwitch).uncheck({ force: true });
        cy.get(selectors.whenAnyCveFixableSwitch).uncheck({ force: true });
        cy.get(selectors.customDateSwitch).uncheck({ force: true });
        cy.get(selectors.indefiniteOptionEnabledSwitch).uncheck({ force: true });

        cy.get(selectors.saveButton).click();
        cy.get('.pf-c-alert:contains("The configuration was updated successfully")');

        // Revisit Workload CVEs and verify that the updated options are available
        visitWorkloadCveOverview();
        selectSingleCveForException('DEFERRAL');
        cy.get('button:contains("Options")').click();

        cy.get('label:contains("For 1 days")');
        cy.get('label:contains("For 2 days")');
        cy.get('label:contains("For 3 days")');
        cy.get('label:contains("For 4 days")');
        cy.get("label:contains('When any CVE is fixable')").should('not.exist');
        cy.get("label:contains('When all CVEs are fixable')").should('not.exist');
        cy.get("label:contains('Until a specific date')").should('not.exist');
        cy.get("label:contains('Indefinitely')").should('not.exist');
    });
});

import dateFns from 'date-fns';

import withAuth from '../helpers/basicAuth';
import { interceptAndOverrideFeatureFlags } from '../helpers/request';
import { visit } from '../helpers/visit';
import {
    credentialForScannerExpiryAlias,
    integrationHealthVulnDefinitionsAlias,
    setClock,
    visitSystemHealth,
    visitSystemHealthWithKeysRemoved,
} from '../helpers/systemHealth';
import { visitIntegrationsDashboard } from './integrations/integrations.helpers';
import { visitSystemConfigurationWithScannerCredentialExpiryBanner } from './credentialExpiry/credentialExpiry.helpers';
import { visitSystemConfiguration } from './systemConfig/systemConfig.helpers';
import { visitNodeCveOverviewPage } from './vulnerabilities/nodeCves/NodeCve.helpers';

const vulnDefinitionsCardSelector =
    '.pf-v6-c-card:contains("StackRox Scanner Vulnerability Definitions")';
const scannerCertificateCardSelector = '.pf-v6-c-card:contains("Scanner certificate")';
const scannerCredentialExpiryBanner = '.pf-v6-c-banner:contains("Scanner certificate")';
const stackRoxScannerTileSelector = '[data-testid="integration-tile"]:contains("StackRox Scanner")';
const nodeCveScannerInfoBoxSelector = '.pf-v6-c-alert:contains("StackRox Scanner")';

const disabledMessage = 'disabled by your administrator';

describe('Legacy Scanner feature flag (ROX_LEGACY_SCANNER)', () => {
    withAuth();

    describe('when disabled', () => {
        beforeEach(() => {
            interceptAndOverrideFeatureFlags({ ROX_LEGACY_SCANNER: false });
        });

        it('should hide StackRox Scanner tile on Integrations page', () => {
            visitIntegrationsDashboard();

            cy.get(stackRoxScannerTileSelector).should('not.exist');
        });

        it('should hide scanner comparison info box on Node CVEs page', () => {
            interceptAndOverrideFeatureFlags({
                ROX_LEGACY_SCANNER: false,
                ROX_SCANNER_V4: true,
                ROX_NODE_INDEX_ENABLED: true,
            });

            visitNodeCveOverviewPage();

            cy.get(nodeCveScannerInfoBoxSelector).should('not.exist');
        });

        it('should hide scanner credential expiry banner', () => {
            visitSystemConfiguration();

            cy.get(scannerCredentialExpiryBanner).should('not.exist');
        });

        it('should show disabled vuln definitions card with message on System Health page', () => {
            visitSystemHealthWithKeysRemoved([
                credentialForScannerExpiryAlias,
                integrationHealthVulnDefinitionsAlias,
            ]);

            cy.get(vulnDefinitionsCardSelector).should('contain', disabledMessage);
        });

        it('should show disabled scanner certificate card with message on System Health page', () => {
            visitSystemHealthWithKeysRemoved([
                credentialForScannerExpiryAlias,
                integrationHealthVulnDefinitionsAlias,
            ]);

            cy.get(scannerCertificateCardSelector).should('contain', disabledMessage);
        });

        it('should show disabled feature page for Platform CVEs', () => {
            visit('/main/vulnerabilities/platform-cves');

            cy.get('h1').should('contain', 'Kubernetes components');
            cy.get('body').should('contain', disabledMessage);
            cy.get('a:contains("Go to Vulnerability Management")').should('exist');
        });
    });

    describe('when enabled', () => {
        beforeEach(() => {
            interceptAndOverrideFeatureFlags({ ROX_LEGACY_SCANNER: true });
        });

        it('should show active vuln definitions card on System Health page', () => {
            const currentDatetime = new Date('2020-12-10T02:04:59.377369440Z');
            const lastUpdatedTimestamp = '2020-12-09T03:04:59.377369440Z';

            const staticResponseMap = {
                [integrationHealthVulnDefinitionsAlias]: {
                    body: { lastUpdatedTimestamp },
                },
            };

            setClock(currentDatetime);
            visitSystemHealth(staticResponseMap);

            cy.get(vulnDefinitionsCardSelector).should('exist');
            cy.get(vulnDefinitionsCardSelector).should('not.have.css', 'opacity', '0.5');
            cy.get(vulnDefinitionsCardSelector).should('contain', 'up to date');
        });

        it('should show active scanner certificate card on System Health page', () => {
            visitSystemHealth();

            cy.get(scannerCertificateCardSelector).should('exist');
            cy.get(scannerCertificateCardSelector).should('not.have.css', 'opacity', '0.5');
        });

        it('should show StackRox Scanner tile on Integrations page', () => {
            visitIntegrationsDashboard();

            cy.get(stackRoxScannerTileSelector).should('exist');
        });

        it('should show scanner credential expiry banner', () => {
            const expiry = dateFns.addDays(new Date(), 2);

            visitSystemConfigurationWithScannerCredentialExpiryBanner(expiry);

            cy.get(scannerCredentialExpiryBanner).should('exist');
        });
    });
});

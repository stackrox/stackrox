import scopeSelectors from '../../helpers/scopeSelectors';

const clusterPageSelector = `#main-page-container:has('a.pf-v6-c-breadcrumb__link:contains("Clusters")')`;

export const selectors = {
    clusterHealth: scopeSelectors(clusterPageSelector, {
        credentialExpiration: '[data-testid="credentialExpiration"]',
        reissueCertificatesLink: '[data-testid="reissueCertificatesLink"]',
        reissueCertificateButton: '[data-testid="reissueCertificateButton"]',
        downloadToReissueCertificate: '[data-testid="downloadToReissueCertificate"]',
        downloadedToReissueCertificate: '[data-testid="downloadedToReissueCertificate"]',
        upgradeToReissueCertificate: '[data-testid="upgradeToReissueCertificate"]',
        upgradedToReissueCertificate: '[data-testid="upgradedToReissueCertificate"]',
        manageTokensButton: '[data-testid="manageTokens"]',
    }),
};

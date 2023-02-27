import scopeSelectors from '../../helpers/scopeSelectors';

export const selectors = {
    clusters: scopeSelectors('[data-testid="clusters-table"]', {
        // Ignore the first checkbox column and last delete column.
        tableDataCell: '.rt-tr-group:not(.hidden) .rt-td:not(:first-child):not(.hidden)',
    }),
    clusterForm: scopeSelectors('[data-testid="cluster-form"]', {
        nameInput: 'input[name="name"]',
    }),
    clusterHealth: scopeSelectors('[data-testid="clusters-side-panel"]', {
        clusterStatus: '[data-testid="clusterStatus"]',
        sensorStatus: '[data-testid="sensorStatus"]',
        collectorStatus: '[data-testid="collectorStatus"]',
        admissionControlStatus: '[data-testid="admissionControlStatus"]',
        admissionControlHealthInfo: scopeSelectors('[data-testid="admissionControlHealthInfo"]', {
            totalReadyPods: '[data-testid="totalReadyPods"]',
            totalDesiredPods: '[data-testid="totalDesiredPods"]',
        }),
        admissionControlInfoComplete: '[data-testid="admissionControlInfoComplete"]',
        collectorHealthInfo: scopeSelectors('[data-testid="collectorHealthInfo"]', {
            totalReadyPods: '[data-testid="totalReadyPods"]',
            totalDesiredPods: '[data-testid="totalDesiredPods"]',
            totalRegisteredNodes: '[data-testid="totalRegisteredNodes"]',
        }),
        collectorInfoComplete: '[data-testid="collectorInfoComplete"]',
        sensorUpgrade: '[data-testid="sensorUpgrade"]',
        sensorVersion: '[data-testid="sensorVersion"]',
        centralVersion: '[data-testid="centralVersion"]',
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

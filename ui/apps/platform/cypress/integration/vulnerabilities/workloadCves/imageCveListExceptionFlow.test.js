import withAuth from '../../../helpers/basicAuth';

import {
    cancelAllCveExceptions,
    fillAndSubmitExceptionForm,
    getDateString,
    getFutureDateByDays,
    selectMultipleCvesForException,
    selectSingleCveForException,
    verifyExceptionConfirmationDetails,
    verifySelectedCvesInModal,
    visitImageSinglePageWithMockedResponses,
} from './WorkloadCves.helpers';

describe('Workload CVE Image page deferral and false positive flows', () => {
    withAuth();

    beforeEach(() => {
        cancelAllCveExceptions();
    });

    after(() => {
        cancelAllCveExceptions();
    });

    it('should defer a single CVE', () => {
        visitImageSinglePageWithMockedResponses().then((image) => {
            selectSingleCveForException('DEFERRAL').then((cveName) => {
                verifySelectedCvesInModal([cveName]);
                fillAndSubmitExceptionForm({
                    comment: 'Test comment',
                    expiryLabel: 'When all CVEs are fixable',
                });
                verifyExceptionConfirmationDetails({
                    expectedAction: 'Deferral',
                    cves: [cveName],
                    scope: `${image}:*`,
                    expiry: 'When all CVEs are fixable',
                });
            });
        });
    });

    // TODO(ROX-27510): CI improvements 2025-02-12: The test is unstable.
    it.skip('should defer multiple selected CVEs', () => {
        visitImageSinglePageWithMockedResponses().then((image) => {
            selectMultipleCvesForException('DEFERRAL').then((cveNames) => {
                verifySelectedCvesInModal(cveNames);
                fillAndSubmitExceptionForm({
                    comment: 'Test comment',
                    expiryLabel: '30 days',
                });

                verifyExceptionConfirmationDetails({
                    expectedAction: 'Deferral',
                    cves: cveNames,
                    scope: `${image}:*`,
                    expiry: `${getDateString(getFutureDateByDays(30))} (30 days)`,
                });
            });
        });
    });

    it('should mark a single CVE as false positive', () => {
        visitImageSinglePageWithMockedResponses().then((image) => {
            selectSingleCveForException('FALSE_POSITIVE').then((cveName) => {
                verifySelectedCvesInModal([cveName]);
                fillAndSubmitExceptionForm({ comment: 'Test comment' });
                verifyExceptionConfirmationDetails({
                    expectedAction: 'False positive',
                    cves: [cveName],
                    scope: `${image}:*`,
                });
            });
        });
    });

    // TODO(ROX-27251): CI improvements 2025-02-12: The test is unstable.
    it.skip('should mark multiple selected CVEs as false positive', () => {
        visitImageSinglePageWithMockedResponses().then((image) => {
            selectMultipleCvesForException('FALSE_POSITIVE').then((cveNames) => {
                verifySelectedCvesInModal(cveNames);
                fillAndSubmitExceptionForm({
                    comment: 'Test comment',
                });
                verifyExceptionConfirmationDetails({
                    expectedAction: 'False positive',
                    cves: cveNames,
                    scope: `${image}:*`,
                });
            });
        });
    });
});

import withAuth from '../../helpers/basicAuth';
import { getInputByLabel } from '../../helpers/formHelpers';

import { triggerScan, visitComplianceDashboard } from './Compliance.helpers';
import {
    clickSaveAndWaitForPatchComplianceStandards,
    openModal,
    selectorForModal,
    selectorInModal,
    selectorInWidget,
} from './hideScanResults.helpers';

const titleAcrossClusters = 'Passing standards across clusters';
const titleAcrossNamespaces = 'Passing standards across namespaces';

const forHIPAA = {
    standardId: 'HIPAA_164',
    barLink: 'a:contains("HIPAA")',
    checkboxLabel: 'HIPAA 164',
};

const forNIST190 = {
    standardId: 'NIST_800_190',
    barLink: 'a:contains("NIST SP 800-190")',
    checkboxLabel: 'NIST SP 800-190',
};

const forNIST53 = {
    standardId: 'NIST_SP_800_53_Rev_4',
    barLink: 'a:contains("NIST SP 800-53")',
    checkboxLabel: 'NIST SP 800-53',
};

function assertExistenceOfStandard(forStandard, existence) {
    const { barLink } = forStandard;
    const existOrNot = existence ? 'exist' : 'not.exist';

    cy.get(selectorInWidget(titleAcrossClusters, barLink)).should(existOrNot);
    cy.get(selectorInWidget(titleAcrossNamespaces, barLink)).should(existOrNot);

    // TODO columns in entity tables
}

function assertCheckedAndClickStandard(forStandard, checked) {
    const { checkboxLabel } = forStandard;
    const haveAttrOrNot = checked ? 'have.attr' : 'not.have.attr';

    getInputByLabel(checkboxLabel).should(haveAttrOrNot, 'checked');
    getInputByLabel(checkboxLabel).click();
}

describe('Compliance hideScanResults', () => {
    withAuth();

    it('should open modal and then cancel', () => {
        triggerScan(); // in case complianceDashboard.test.js is skipped
        openModal();

        cy.get(selectorInModal('button:contains("Save")')).should('be.disabled');
        cy.get(selectorInModal('button:contains("Cancel")')).click();
        cy.get(selectorForModal).should('not.exist');
    });

    it('should hide HIPAA standard', () => {
        visitComplianceDashboard();

        assertExistenceOfStandard(forHIPAA, true);

        openModal();

        assertCheckedAndClickStandard(forHIPAA, true);
        clickSaveAndWaitForPatchComplianceStandards([forHIPAA.standardId]);

        assertExistenceOfStandard(forHIPAA, false);
    });

    // The following test depends on the preceding test.

    it('should show HIPAA standard and hide NIST standards', () => {
        visitComplianceDashboard();

        assertExistenceOfStandard(forHIPAA, false);
        assertExistenceOfStandard(forNIST190, true);
        assertExistenceOfStandard(forNIST53, true);

        openModal();
        assertCheckedAndClickStandard(forHIPAA, false);
        assertCheckedAndClickStandard(forNIST190, true);
        assertCheckedAndClickStandard(forNIST53, true);

        clickSaveAndWaitForPatchComplianceStandards([
            forHIPAA.standardId,
            forNIST190.standardId,
            forNIST53.standardId,
        ]);

        assertExistenceOfStandard(forHIPAA, true);
        assertExistenceOfStandard(forNIST190, false);
        assertExistenceOfStandard(forNIST53, false);
    });

    // The following test depends on the preceding test.

    it('should show NIST standards', () => {
        visitComplianceDashboard();

        assertExistenceOfStandard(forNIST190, false);
        assertExistenceOfStandard(forNIST53, false);

        openModal();
        assertCheckedAndClickStandard(forNIST190, false);
        assertCheckedAndClickStandard(forNIST53, false);

        clickSaveAndWaitForPatchComplianceStandards([forNIST190.standardId, forNIST53.standardId]);

        assertExistenceOfStandard(forNIST190, true);
        assertExistenceOfStandard(forNIST53, true);
    });

    // hideScanResults is same at the end as it was at the beginning
});

import { selectors as riskSelectors, url as riskUrl } from '../../constants/RiskPage';
import { selectors as policySelectors } from '../../constants/PoliciesPage';
import { url as policiesUrl } from '../../constants/PoliciesPagePatternFly';
import { hasFeatureFlag } from '../../helpers/features';
import withAuth from '../../helpers/basicAuth';

describe('Risk search to new policy', () => {
    withAuth();

    const navigateToPolicy = (url) => {
        cy.visit(url);

        cy.get(riskSelectors.createPolicyButton).click();

        cy.location().should((location) => {
            expect(location.pathname).to.eq(policiesUrl);
        });
        cy.get(policySelectors.nextButton).click();
    };

    before(function beforeHook() {
        if (hasFeatureFlag('ROX_POLICIES_PATTERNFLY')) {
            this.skip();
        }
    });

    it('should create a policy with a multiselect field, like Add Capabilities', () => {
        navigateToPolicy(`${riskUrl}?s[Add%20Capabilities]=NET_BIND_SERVICE`);

        cy.get(`${policySelectors.booleanPolicySection.policyFieldCard}:first`).should(
            'contain.text',
            'Add Capabilities:'
        );
        cy.get(
            `${policySelectors.booleanPolicySection.policyFieldCard}:first .react-select__single-value`
        ).should('contain.text', 'NET_BIND_SERVICE');
    });

    it('should create a policy with a numeric comparison criterion, like CPU cores limit', () => {
        navigateToPolicy(`${riskUrl}?s[CPU%20Cores%20Limit]=2`);

        cy.get(`${policySelectors.booleanPolicySection.policyFieldCard}:first`).should(
            'contain.text',
            'Container CPU Limit:'
        );
        cy.get(
            `${policySelectors.booleanPolicySection.policyFieldCard}:first .react-select__single-value`
        ).should('contain.text', 'Is equal to');
        cy.get(`${policySelectors.booleanPolicySection.policyFieldCard}:first input:last`).should(
            'contain.value',
            '2'
        );
    });

    it('should create a policy with a key/value criterion with only the key specified, like Dockerfile Instruction key', () => {
        navigateToPolicy(`${riskUrl}?s[Dockerfile%20Instruction%20Keyword]=RUN`);

        cy.get(`${policySelectors.booleanPolicySection.policyFieldCard}:first .uppercase`).should(
            'include.text',
            'Disallowed dockerfile line:'
        );
        cy.get(
            `${policySelectors.booleanPolicySection.policyFieldCard}:first .react-select__single-value`
        ).should('contain.text', 'RUN');
    });

    it('should create a policy with a key/value criterion with only the value specified, like Dockerfile Instruction value', () => {
        navigateToPolicy(`${riskUrl}?s[Dockerfile%20Instruction%20Value]=%5B"%2Fbin%2Fsh"%5D`);

        cy.get(`${policySelectors.booleanPolicySection.policyFieldCard}:first .uppercase`).should(
            'include.text',
            'Disallowed dockerfile line:'
        );
        cy.get(`${policySelectors.booleanPolicySection.policyFieldCard}:first input:last`).should(
            'contain.value',
            '["/bin/sh"]'
        );
    });

    it('should create a policy with correct Cluster and Namespace and Label scopes', () => {
        cy.visit(
            `${riskUrl}?s[Cluster]=remote&s[Namespace]=docker&s[Label]=com.docker.deploy-namespace%3Ddocker`
        );

        cy.get(riskSelectors.createPolicyButton).click();

        cy.location().should((location) => {
            expect(location.pathname).to.eq(policiesUrl);
        });
        cy.get('.react-select__single-value:contains("remote")');
        cy.get('input[placeholder="Namespace"]').should('contain.value', 'docker');
        cy.get('input[placeholder="Label Key"]').should(
            'contain.value',
            'com.docker.deploy-namespace'
        );
        cy.get('input[placeholder="Label Value"]').should('contain.value', 'docker');
    });

    it('should not create a policy for a search with invalid search criteria', () => {
        cy.visit(`${riskUrl}?s[Add%20Capability]=NONEXISTENT`);

        cy.get(riskSelectors.createPolicyButton).click();

        cy.location().should((location) => {
            expect(location.pathname).to.eq(riskUrl);
        });
        cy.get(policySelectors.toast).contains('Could not create a policy from this search');
    });
});

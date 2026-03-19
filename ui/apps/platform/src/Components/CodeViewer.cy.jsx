import CodeViewer from './CodeViewer';

const sampleYaml = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  creationTimestamp: "2025-04-10T14:44:09Z"
  labels:
    network-policy-generator.stackrox.io/generated: "true"
  name: stackrox-generated-nginx
  namespace: cypress-test
spec:
  podSelector:
    matchLabels:
      app: nginx
  policyTypes:
  - Ingress

`;

describe(Cypress.spec.relative, () => {
    it('should display correctly numbered lines', () => {
        cy.mount(<CodeViewer code={sampleYaml} />);

        sampleYaml.split('\n').forEach((lineText, index) => {
            cy.get(`code span:contains('${index + 1}'):contains('${lineText}')`).should('exist');
        });
    });

    it('should copy code snippet to the clipboard', () => {
        cy.mount(<CodeViewer code={sampleYaml} />);

        cy.get('button[aria-label="Copy code to clipboard"]').click();

        cy.window().then((window) => {
            window.navigator.clipboard.readText().then((text) => {
                expect(text).to.eq(sampleYaml);
            });
        });
    });
});

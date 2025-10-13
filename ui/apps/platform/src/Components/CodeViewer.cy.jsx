import React from 'react';
import CodeViewer, { CodeViewerThemeProvider } from './CodeViewer';

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

    it('should toggle light and dark editor themes', () => {
        cy.mount(<CodeViewer code={sampleYaml} />);

        // Default to light mode
        cy.get('.pf-v5-theme-dark').should('not.exist');

        cy.get('button[aria-label="Set dark theme"]').click();
        cy.get('.pf-v5-theme-dark');

        cy.get('button[aria-label="Set light theme"]').click();
        cy.get('.pf-v5-theme-dark').should('not.exist');
    });

    it('should share theme state across multiple instances', () => {
        cy.mount(
            <CodeViewerThemeProvider>
                <CodeViewer code={sampleYaml} />
                <CodeViewer code={sampleYaml} />
            </CodeViewerThemeProvider>
        );

        cy.get('.pf-v5-theme-dark').should('not.exist');

        cy.get('button[aria-label="Set dark theme"]').eq(0);
        cy.get('button[aria-label="Set dark theme"]').eq(1);
        cy.get('button[aria-label="Set dark theme"]').eq(0).click();

        cy.get('.pf-v5-theme-dark').should('have.length', 2);

        cy.get('button[aria-label="Set light theme"]').eq(0);
        cy.get('button[aria-label="Set light theme"]').eq(1);
        cy.get('button[aria-label="Set light theme"]').eq(1).click();

        cy.get('.pf-v5-theme-dark').should('not.exist');
    });
});

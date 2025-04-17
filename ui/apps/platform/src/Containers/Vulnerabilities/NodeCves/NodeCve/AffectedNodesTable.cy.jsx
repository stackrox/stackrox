import React from 'react';

import ComponentTestProviders from 'test-utils/ComponentProviders';
import AffectedNodesTable from './AffectedNodesTable';

function mockNodeVulnerability(fields) {
    return {
        vulnerabilityId: '',
        cve: '',
        severity: 'MODERATE_VULNERABILITY_SEVERITY',
        isFixable: false,
        cvss: 2.5,
        scoreVersion: 'V3',
        fixedByVersion: '',
        ...fields,
    };
}

function mockNodeComponent(fields) {
    return {
        name: 'podman',
        version: '3:4.4.1-21.rhaos4.15.el9.x86_64',
        fixedIn: '',
        source: 'INFRASTRUCTURE',
        nodeVulnerabilities: [mockNodeVulnerability()],
        ...fields,
    };
}

function mockNode(fields) {
    return {
        id: '',
        name: 'node name',
        nodeComponents: [mockNodeComponent()],
        cluster: {
            name: 'test-cluster',
        },
        osImage: 'RHEL',
        ...fields,
    };
}

function setup(tableState) {
    cy.mount(
        <ComponentTestProviders>
            <AffectedNodesTable
                tableState={tableState}
                getSortParams={() => {}}
                onClearFilters={() => {}}
            />
        </ComponentTestProviders>
    );
}

describe(Cypress.spec.relative, () => {
    describe('when the table is in a non-success state', () => {
        it('should render a spinner when loading', () => {
            setup({ type: 'LOADING' });

            cy.findByRole('progressbar');
        });

        it('should render an error message when errored', () => {
            setup({ type: 'ERROR', error: new Error('An error from Cypress') });

            cy.findByText('An error from Cypress');
            cy.findByText('An error has occurred. Try clearing any filters or refreshing the page');
        });

        it('should render a message when no data is available', () => {
            setup({ type: 'EMPTY' });

            cy.findByText('No results found');
            cy.findByText('There are no nodes that are affected by this CVE');
        });

        it('should render a message when a filter results in no data', () => {
            setup({ type: 'FILTERED_EMPTY' });

            cy.findByText('No results found');
            cy.findByRole('button', { name: 'Clear filters' });
        });
    });

    describe('when the table is in a success state', () => {
        it('should render the correct number of rows', () => {
            const nodes = [
                mockNode({ id: `${1}`, name: `${1}-name` }),
                mockNode({ id: `${2}`, name: `${2}-name` }),
                mockNode({ id: `${3}`, name: `${3}-name` }),
                mockNode({ id: `${4}`, name: `${4}-name` }),
            ];
            const rowCount = nodes.length;
            const headerCount = 1;

            setup({ type: 'COMPLETE', data: nodes });

            cy.findAllByRole('row').should('have.length', headerCount + rowCount);
        });

        it('should correctly render the severity of a node with multiple components affected', () => {
            let id = 0;
            // Accepts an array of arrays of severities, where the outer array represents the components
            // and the inner array represents the severities of vulnerabilities for that component
            function createNodeWithComponentSeverities(severityGroups) {
                id += 1;
                return mockNode({
                    id,
                    nodeComponents: severityGroups.map((severities) =>
                        mockNodeComponent({
                            nodeVulnerabilities: severities.map((severity) =>
                                mockNodeVulnerability({ severity })
                            ),
                        })
                    ),
                });
            }

            // Create nodes with mixed severities across affected components
            const lowSeverityNode = createNodeWithComponentSeverities([
                ['LOW_VULNERABILITY_SEVERITY'],
            ]);
            const moderateSeverityNode = createNodeWithComponentSeverities([
                ['LOW_VULNERABILITY_SEVERITY'],
                ['BOGUS_VALUE'],
                [
                    'LOW_VULNERABILITY_SEVERITY',
                    'MODERATE_VULNERABILITY_SEVERITY',
                    'LOW_VULNERABILITY_SEVERITY',
                ],
            ]);
            const importantSeverityNode = createNodeWithComponentSeverities([
                [
                    'MODERATE_VULNERABILITY_SEVERITY',
                    'IMPORTANT_VULNERABILITY_SEVERITY',
                    'IMPORTANT_VULNERABILITY_SEVERITY',
                    'BOGUS_VALUE',
                ],
                ['MODERATE_VULNERABILITY_SEVERITY', 'MODERATE_VULNERABILITY_SEVERITY'],
            ]);
            const criticalSeverityNode = createNodeWithComponentSeverities([
                [
                    'UNKNOWN_VULNERABILITY_SEVERITY',
                    'LOW_VULNERABILITY_SEVERITY',
                    'BOGUS_VALUE',
                    'CRITICAL_VULNERABILITY_SEVERITY',
                    'BOGUS_VALUE',
                    'MODERATE_VULNERABILITY_SEVERITY',
                    'IMPORTANT_VULNERABILITY_SEVERITY',
                ],
            ]);
            const unknownSeverityNode = createNodeWithComponentSeverities([
                ['UNKNOWN_VULNERABILITY_SEVERITY'],
            ]);

            const nodes = [
                lowSeverityNode,
                moderateSeverityNode,
                importantSeverityNode,
                criticalSeverityNode,
                unknownSeverityNode,
            ];

            setup({ type: 'COMPLETE', data: nodes });

            // Avoid the 0th row which is the header, and then check that the top severity is rendered correctly
            // for each row
            [
                { rowIndex: 1, expectedSeverity: 'Low' },
                { rowIndex: 2, expectedSeverity: 'Moderate' },
                { rowIndex: 3, expectedSeverity: 'Important' },
                { rowIndex: 4, expectedSeverity: 'Critical' },
                { rowIndex: 5, expectedSeverity: 'Unknown' },
            ].forEach(({ rowIndex, expectedSeverity }) => {
                cy.findAllByRole('row')
                    .eq(rowIndex)
                    .within(() => {
                        cy.get(`td[data-label="CVE severity"]:contains("${expectedSeverity}")`);
                    });
            });
        });

        it('should correctly render the fixability of a node with multiple components affected', () => {
            let id = 0;
            function createNodeWithComponentFixedIn(fixedByGroups) {
                id += 1;
                return mockNode({
                    id,
                    nodeComponents: fixedByGroups.map((fixedByVersions) =>
                        mockNodeComponent({
                            nodeVulnerabilities: fixedByVersions.map((fixedByVersion) =>
                                mockNodeVulnerability({ fixedByVersion })
                            ),
                        })
                    ),
                });
            }

            const fixedByVersion = '3:4.4.1-21.rhaos4.15.el9.x86_64';
            const notFixedInVersion = '';

            // Create nodes with mixed fixability across affected components
            const nodes = [
                createNodeWithComponentFixedIn([[fixedByVersion]]),
                createNodeWithComponentFixedIn([[notFixedInVersion]]),
                createNodeWithComponentFixedIn([[fixedByVersion, notFixedInVersion]]),
                createNodeWithComponentFixedIn([[fixedByVersion], [notFixedInVersion]]),
            ];

            setup({ type: 'COMPLETE', data: nodes });

            // Avoid the 0th row which is the header, and then check that the fixability is rendered correctly
            // for each row. If any vulnerability for any component is fixable, the node should render fixable.
            [
                { rowIndex: 1, expectedFixability: 'Fixable' },
                { rowIndex: 2, expectedFixability: 'Not fixable' },
                { rowIndex: 3, expectedFixability: 'Fixable' },
                { rowIndex: 4, expectedFixability: 'Fixable' },
            ].forEach(({ rowIndex, expectedFixability }) => {
                cy.findAllByRole('row')
                    .eq(rowIndex)
                    .within(() => {
                        cy.get(`td[data-label="CVE status"]:contains("${expectedFixability}")`);
                    });
            });
        });

        it('should render the highest CVSS score of all matching component vulnerabilities', () => {
            let id = 0;
            function createNodeWithComponentCvss(cvssGroups) {
                id += 1;
                return mockNode({
                    id,
                    nodeComponents: cvssGroups.map((cvssGroup) =>
                        mockNodeComponent({
                            nodeVulnerabilities: cvssGroup.map(mockNodeVulnerability),
                        })
                    ),
                });
            }

            const nodes = [
                createNodeWithComponentCvss([
                    [{ cvss: 2.0, scoreVersion: 'V3' }],
                    [
                        { cvss: 1.0, scoreVersion: 'V3' },
                        { cvss: 1.0, scoreVersion: 'V2' },
                    ],
                ]),
                createNodeWithComponentCvss([
                    [
                        { cvss: 3.0, scoreVersion: 'V3' },
                        { cvss: 4.0, scoreVersion: 'V2' },
                    ],
                ]),
                createNodeWithComponentCvss([
                    [
                        { cvss: 5.0, scoreVersion: 'V2' },
                        { cvss: 6.0, scoreVersion: 'V3' },
                        { cvss: 7.0, scoreVersion: 'V2' },
                    ],
                    [
                        { cvss: 6.0, scoreVersion: 'V3' },
                        { cvss: 2.0, scoreVersion: 'V2' },
                    ],
                ]),
            ];

            setup({ type: 'COMPLETE', data: nodes });

            // Avoid the 0th row which is the header, and then check that the highest CVSS score is rendered correctly
            // for each row
            [
                { rowIndex: 1, expectedCVSS: '2.0', expectedVersion: 'V3' },
                { rowIndex: 2, expectedCVSS: '4.0', expectedVersion: 'V2' },
                { rowIndex: 3, expectedCVSS: '7.0', expectedVersion: 'V2' },
            ].forEach(({ rowIndex, expectedCVSS, expectedVersion }) => {
                cy.findAllByRole('row')
                    .eq(rowIndex)
                    .within(() => {
                        cy.get(
                            `td[data-label="CVSS"]:contains("${expectedCVSS} (${expectedVersion})")`
                        );
                    });
            });
        });

        it('should allow expanding and collapsing the nested component table', () => {
            // TODO: Implement this test
        });
    });
});

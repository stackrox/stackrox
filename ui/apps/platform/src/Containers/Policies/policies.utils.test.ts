import { ClientPolicy, Policy } from 'types/policy.proto';
import { getClientWizardPolicy, getServerPolicy } from './policies.utils';

describe('policies.utils', () => {
    describe('getClientWizardPolicy', () => {
        test('should return a client policy object from a server policy object', () => {
            const serverPolicy: Policy = {
                id: 'e73359bd-68d0-48d6-8e3c-f81cf85e2574',
                name: 'Test policy',
                description: 'a description',
                rationale: 'Rationale here',
                remediation: 'Guidance here',
                disabled: false,
                categories: ['Cryptocurrency Mining', 'System Modification'],
                lifecycleStages: ['BUILD', 'DEPLOY'],
                eventSource: 'NOT_APPLICABLE',
                exclusions: [
                    {
                        name: '',
                        deployment: {
                            name: 'archlinux',
                            scope: {
                                cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                                namespace: 'kube-*',
                                label: { key: 'app', value: 'archlinux' },
                            },
                        },
                        expiration: null,
                    },
                    {
                        name: '',
                        image: { name: 'docker.io/library/archlinux:latest' },
                        expiration: null,
                    },
                    {
                        name: '',
                        image: { name: 'docker.io/library/ghost:latest' },
                        expiration: null,
                    },
                ],
                scope: [
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing',
                        label: { key: 'app', value: 'include1' },
                    },
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing2',
                        label: { key: 'app', value: 'include2' },
                    },
                ],
                severity: 'LOW_SEVERITY',
                enforcementActions: [],
                notifiers: ['10a830c7-dc0b-4d9e-9505-4ae3b72d6b50'],
                lastUpdated: '2024-08-08T19:27:43.987955873Z',
                SORTName: 'Test policy',
                SORTLifecycleStage: 'BUILD,DEPLOY',
                SORTEnforcement: false,
                policyVersion: '1.1',
                policySections: [
                    {
                        sectionName: 'Policy Section 1',
                        policyGroups: [
                            {
                                fieldName: 'Dockerfile Line',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'ENV=ENV=myapp=test' }, { value: 'USER=root' }],
                            },
                            {
                                fieldName: 'Image Signature Verified By',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    {
                                        value: 'io.stackrox.signatureintegration.bef8ab45-2f06-4937-9a97-5c8b5b049f54',
                                    },
                                ],
                            },
                        ],
                    },
                    {
                        sectionName: 'Policy Section 2',
                        policyGroups: [
                            {
                                fieldName: 'Image Remote',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'library/nginx' }],
                            },
                            {
                                fieldName: 'CVSS',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: '>=5' }],
                            },
                        ],
                    },
                ],
                mitreAttackVectors: [
                    { tactic: 'TA0002', techniques: ['T1053.003'] },
                    { tactic: 'TA0004', techniques: ['T1037.003', 'T1037.001'] },
                ],
                criteriaLocked: false,
                mitreVectorsLocked: false,
                isDefault: false,
            };

            const clientPolicy: ClientPolicy = {
                id: 'e73359bd-68d0-48d6-8e3c-f81cf85e2574',
                name: 'Test policy',
                description: 'a description',
                rationale: 'Rationale here',
                remediation: 'Guidance here',
                disabled: false,
                categories: ['Cryptocurrency Mining', 'System Modification'],
                lifecycleStages: ['BUILD', 'DEPLOY'],
                eventSource: 'NOT_APPLICABLE',
                exclusions: [
                    {
                        name: '',
                        deployment: {
                            name: 'archlinux',
                            scope: {
                                cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                                namespace: 'kube-*',
                                label: { key: 'app', value: 'archlinux' },
                            },
                        },
                        expiration: null,
                    },
                    {
                        name: '',
                        image: { name: 'docker.io/library/archlinux:latest' },
                        expiration: null,
                    },
                    {
                        name: '',
                        image: { name: 'docker.io/library/ghost:latest' },
                        expiration: null,
                    },
                ],
                scope: [
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing',
                        label: { key: 'app', value: 'include1' },
                    },
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing2',
                        label: { key: 'app', value: 'include2' },
                    },
                ],
                severity: 'LOW_SEVERITY',
                enforcementActions: [],
                notifiers: ['10a830c7-dc0b-4d9e-9505-4ae3b72d6b50'],
                lastUpdated: '2024-08-08T19:27:43.987955873Z',
                SORTName: 'Test policy',
                SORTLifecycleStage: 'BUILD,DEPLOY',
                SORTEnforcement: false,
                policyVersion: '1.1',
                policySections: [
                    {
                        sectionName: 'Policy Section 1',
                        policyGroups: [
                            {
                                fieldName: 'Dockerfile Line',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    { value: 'ENV=ENV=myapp=test' },
                                    { key: 'USER', value: 'root' },
                                ],
                            },
                            {
                                fieldName: 'Image Signature Verified By',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    {
                                        arrayValue: [
                                            'io.stackrox.signatureintegration.bef8ab45-2f06-4937-9a97-5c8b5b049f54',
                                        ],
                                    },
                                ],
                            },
                        ],
                    },
                    {
                        sectionName: 'Policy Section 2',
                        policyGroups: [
                            {
                                fieldName: 'Image Remote',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'library/nginx' }],
                            },
                            {
                                fieldName: 'CVSS',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ key: '>=', value: '5' }],
                            },
                        ],
                    },
                ],
                mitreAttackVectors: [
                    { tactic: 'TA0002', techniques: ['T1053.003'] },
                    { tactic: 'TA0004', techniques: ['T1037.003', 'T1037.001'] },
                ],
                criteriaLocked: false,
                mitreVectorsLocked: false,
                isDefault: false,
                excludedImageNames: [
                    'docker.io/library/archlinux:latest',
                    'docker.io/library/ghost:latest',
                ],
                excludedDeploymentScopes: [
                    {
                        name: 'archlinux',
                        scope: {
                            cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                            namespace: 'kube-*',
                            label: { key: 'app', value: 'archlinux' },
                        },
                    },
                ],
                serverPolicySections: [
                    {
                        sectionName: 'Policy Section 1',
                        policyGroups: [
                            {
                                fieldName: 'Dockerfile Line',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'ENV=ENV=myapp=test' }, { value: 'USER=root' }],
                            },
                            {
                                fieldName: 'Image Signature Verified By',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    {
                                        value: 'io.stackrox.signatureintegration.bef8ab45-2f06-4937-9a97-5c8b5b049f54',
                                    },
                                ],
                            },
                        ],
                    },
                    {
                        sectionName: 'Policy Section 2',
                        policyGroups: [
                            {
                                fieldName: 'Image Remote',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'library/nginx' }],
                            },
                            {
                                fieldName: 'CVSS',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: '>=5' }],
                            },
                        ],
                    },
                ],
            };

            expect(getClientWizardPolicy(serverPolicy)).toEqual(clientPolicy);
        });
    });

    describe('getServerPolicy', () => {
        test('should return a server policy object from a client policy object', () => {
            const serverPolicy: Policy = {
                id: 'e73359bd-68d0-48d6-8e3c-f81cf85e2574',
                name: 'Test policy',
                description: 'a description',
                rationale: 'Rationale here',
                remediation: 'Guidance here',
                disabled: false,
                categories: ['Cryptocurrency Mining', 'System Modification'],
                lifecycleStages: ['BUILD', 'DEPLOY'],
                eventSource: 'NOT_APPLICABLE',
                exclusions: [
                    {
                        deployment: {
                            name: 'archlinux',
                            scope: {
                                cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                                namespace: 'kube-*',
                                label: { key: 'app', value: 'archlinux' },
                            },
                        },
                    },
                    { image: { name: 'docker.io/library/archlinux:latest' } },
                    { image: { name: 'docker.io/library/ghost:latest' } },
                ],
                scope: [
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing',
                        label: { key: 'app', value: 'include1' },
                    },
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing2',
                        label: { key: 'app', value: 'include2' },
                    },
                ],
                severity: 'LOW_SEVERITY',
                enforcementActions: [],
                notifiers: ['10a830c7-dc0b-4d9e-9505-4ae3b72d6b50'],
                lastUpdated: '2024-08-08T19:27:43.987955873Z',
                SORTName: 'Test policy',
                SORTLifecycleStage: 'BUILD,DEPLOY',
                SORTEnforcement: false,
                policyVersion: '1.1',
                policySections: [
                    {
                        sectionName: 'Policy Section 1',
                        policyGroups: [
                            {
                                fieldName: 'Dockerfile Line',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'ENV=ENV=myapp=test' }, { value: 'USER=root' }],
                            },
                            {
                                fieldName: 'Image Signature Verified By',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    {
                                        value: 'io.stackrox.signatureintegration.bef8ab45-2f06-4937-9a97-5c8b5b049f54',
                                    },
                                ],
                            },
                        ],
                    },
                    {
                        sectionName: 'Policy Section 2',
                        policyGroups: [
                            {
                                fieldName: 'Image Remote',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'library/nginx' }],
                            },
                            {
                                fieldName: 'CVSS',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: '>=5' }],
                            },
                        ],
                    },
                ],
                mitreAttackVectors: [
                    { tactic: 'TA0002', techniques: ['T1053.003'] },
                    { tactic: 'TA0004', techniques: ['T1037.003', 'T1037.001'] },
                ],
                criteriaLocked: false,
                mitreVectorsLocked: false,
                isDefault: false,
            };

            const clientPolicy: ClientPolicy = {
                id: 'e73359bd-68d0-48d6-8e3c-f81cf85e2574',
                name: 'Test policy',
                description: 'a description',
                rationale: 'Rationale here',
                remediation: 'Guidance here',
                disabled: false,
                categories: ['Cryptocurrency Mining', 'System Modification'],
                lifecycleStages: ['BUILD', 'DEPLOY'],
                eventSource: 'NOT_APPLICABLE',
                exclusions: [
                    {
                        name: '',
                        deployment: {
                            name: 'archlinux',
                            scope: {
                                cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                                namespace: 'kube-*',
                                label: { key: 'app', value: 'archlinux' },
                            },
                        },
                        expiration: null,
                    },
                    {
                        name: '',
                        image: { name: 'docker.io/library/archlinux:latest' },
                        expiration: null,
                    },
                    {
                        name: '',
                        image: { name: 'docker.io/library/ghost:latest' },
                        expiration: null,
                    },
                ],
                scope: [
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing',
                        label: { key: 'app', value: 'include1' },
                    },
                    {
                        cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                        namespace: 'ui-testing2',
                        label: { key: 'app', value: 'include2' },
                    },
                ],
                severity: 'LOW_SEVERITY',
                enforcementActions: [],
                notifiers: ['10a830c7-dc0b-4d9e-9505-4ae3b72d6b50'],
                lastUpdated: '2024-08-08T19:27:43.987955873Z',
                SORTName: 'Test policy',
                SORTLifecycleStage: 'BUILD,DEPLOY',
                SORTEnforcement: false,
                policyVersion: '1.1',
                policySections: [
                    {
                        sectionName: 'Policy Section 1',
                        policyGroups: [
                            {
                                fieldName: 'Dockerfile Line',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    { value: 'ENV=ENV=myapp=test' },
                                    { key: 'USER', value: 'root' },
                                ],
                            },
                            {
                                fieldName: 'Image Signature Verified By',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    {
                                        arrayValue: [
                                            'io.stackrox.signatureintegration.bef8ab45-2f06-4937-9a97-5c8b5b049f54',
                                        ],
                                    },
                                ],
                            },
                        ],
                    },
                    {
                        sectionName: 'Policy Section 2',
                        policyGroups: [
                            {
                                fieldName: 'Image Remote',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'library/nginx' }],
                            },
                            {
                                fieldName: 'CVSS',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ key: '>=', value: '5' }],
                            },
                        ],
                    },
                ],
                mitreAttackVectors: [
                    { tactic: 'TA0002', techniques: ['T1053.003'] },
                    { tactic: 'TA0004', techniques: ['T1037.003', 'T1037.001'] },
                ],
                criteriaLocked: false,
                mitreVectorsLocked: false,
                isDefault: false,
                excludedImageNames: [
                    'docker.io/library/archlinux:latest',
                    'docker.io/library/ghost:latest',
                ],
                excludedDeploymentScopes: [
                    {
                        name: 'archlinux',
                        scope: {
                            cluster: '5c5c9aae-9c92-4648-88a2-9e593c225fa1',
                            namespace: 'kube-*',
                            label: { key: 'app', value: 'archlinux' },
                        },
                    },
                ],
                serverPolicySections: [
                    {
                        sectionName: 'Policy Section 1',
                        policyGroups: [
                            {
                                fieldName: 'Dockerfile Line',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'ENV=ENV=myapp=test' }, { value: 'USER=root' }],
                            },
                            {
                                fieldName: 'Image Signature Verified By',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [
                                    {
                                        value: 'io.stackrox.signatureintegration.bef8ab45-2f06-4937-9a97-5c8b5b049f54',
                                    },
                                ],
                            },
                        ],
                    },
                    {
                        sectionName: 'Policy Section 2',
                        policyGroups: [
                            {
                                fieldName: 'Image Remote',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: 'library/nginx' }],
                            },
                            {
                                fieldName: 'CVSS',
                                booleanOperator: 'OR',
                                negate: false,
                                values: [{ value: '>=5' }],
                            },
                        ],
                    },
                ],
            };

            expect(getServerPolicy(clientPolicy)).toEqual(serverPolicy);
        });
    });
});

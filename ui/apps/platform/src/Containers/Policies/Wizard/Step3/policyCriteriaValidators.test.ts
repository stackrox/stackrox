import { policyEventSources } from 'types/policy.proto';
import type { ClientPolicyGroup, ClientPolicySection } from 'types/policy.proto';

import { auditLogDescriptor } from './policyCriteriaDescriptors';
import { policySectionValidators } from './policyCriteriaValidators';
import type { PolicyContext } from './policyCriteriaValidators';

function mockCriterionWithName(name: string, values: { value: string }[] = []): ClientPolicyGroup {
    return {
        fieldName: name,
        booleanOperator: 'OR',
        negate: false,
        values,
    };
}

describe('policyCriteriaValidators', () => {
    describe('policySectionValidators registry', () => {
        it('should have unique validator names', () => {
            const names = policySectionValidators.map((v) => v.name);
            const uniqueNames = new Set(names);
            expect(uniqueNames.size).toBe(names.length);
        });
    });

    describe('Audit log required fields validator', () => {
        const validator = policySectionValidators.find(
            (v) => v.name === 'Audit log required fields'
        );

        if (!validator) {
            throw new Error('Audit log required fields validator not found');
        }

        const context: PolicyContext = {
            eventSource: 'AUDIT_LOG_EVENT',
            lifecycleStages: ['RUNTIME'],
        };

        it('should only apply to AUDIT_LOG_EVENT event source regardless of lifecycle stages', () => {
            policyEventSources.forEach((eventSource) => {
                expect(
                    validator.appliesTo({
                        eventSource,
                        lifecycleStages: ['RUNTIME'],
                    })
                ).toBe(eventSource === 'AUDIT_LOG_EVENT');
            });
        });

        it('should fail with missing other criterion error when only one required criterion is present', () => {
            auditLogDescriptor.forEach((descriptor) => {
                const { name } = descriptor;
                const section: ClientPolicySection = {
                    sectionName: 'Test Section',
                    policyGroups: [mockCriterionWithName(name)],
                };
                const error = validator.validate(section, context);

                if (name === 'Kubernetes Resource') {
                    expect(error).toContain('Kubernetes API verb');
                    expect(error).not.toContain('Kubernetes resource type');
                } else if (name === 'Kubernetes API Verb') {
                    expect(error).toContain('Kubernetes resource type');
                    expect(error).not.toContain('Kubernetes API verb');
                } else {
                    expect(error).toContain('Kubernetes API verb');
                    expect(error).toContain('Kubernetes resource type');
                }
            });
        });

        it('should pass when just the required criteria are present', () => {
            const section: ClientPolicySection = {
                sectionName: 'Test Section',
                policyGroups: [
                    mockCriterionWithName('Kubernetes Resource'),
                    mockCriterionWithName('Kubernetes API Verb'),
                ],
            };
            expect(validator.validate(section, context)).toBeUndefined();
        });

        it('should pass when all required criteria are present for all audit log criteria', () => {
            const nonRequiredDescriptors = auditLogDescriptor.filter(
                (d) => d.name !== 'Kubernetes Resource' && d.name !== 'Kubernetes API Verb'
            );

            nonRequiredDescriptors.forEach((descriptor) => {
                const section: ClientPolicySection = {
                    sectionName: 'Test Section',
                    policyGroups: [
                        mockCriterionWithName('Kubernetes Resource'),
                        mockCriterionWithName('Kubernetes API Verb'),
                        mockCriterionWithName(descriptor.name),
                    ],
                };
                expect(validator.validate(section, context)).toBeUndefined();
            });
        });

        it('should fail when section has no policy groups', () => {
            const section: ClientPolicySection = {
                sectionName: 'Test Section',
                policyGroups: [],
            };
            const error = validator.validate(section, context);
            expect(error).toBeDefined();
        });
    });

    describe('File operation requires mounted file path (Deploy) validator', () => {
        const validator = policySectionValidators.find(
            (v) => v.name === 'File operation requires file path (Deploy)'
        );

        if (!validator) {
            throw new Error('File operation requires file path (Deploy) validator not found');
        }

        const context: PolicyContext = {
            eventSource: 'DEPLOYMENT_EVENT',
            lifecycleStages: ['RUNTIME'],
        };

        it('should only apply to DEPLOYMENT_EVENT with RUNTIME lifecycle stage', () => {
            policyEventSources.forEach((eventSource) => {
                expect(
                    validator.appliesTo({
                        eventSource,
                        lifecycleStages: ['RUNTIME'],
                    })
                ).toBe(eventSource === 'DEPLOYMENT_EVENT');
            });
        });

        it('should fail when File Operation is present but Effective Path is missing', () => {
            const section: ClientPolicySection = {
                sectionName: 'Test Section',
                policyGroups: [mockCriterionWithName('File Operation', [{ value: 'CREATE' }])],
            };
            const error = validator.validate(section, context);
            expect(error).toBeDefined();
        });

        it('should pass when File Operation and Effective Path both present with values', () => {
            const section: ClientPolicySection = {
                sectionName: 'Test Section',
                policyGroups: [
                    mockCriterionWithName('File Operation', [{ value: 'CREATE' }]),
                    mockCriterionWithName('Effective Path', [{ value: '/etc/passwd' }]),
                ],
            };
            expect(validator.validate(section, context)).toBeUndefined();
        });
    });

    describe('File operation requires node file path (Node) validator', () => {
        const validator = policySectionValidators.find(
            (v) => v.name === 'File operation requires file path (Node)'
        );

        if (!validator) {
            throw new Error('File operation requires file path (Node) validator not found');
        }

        const context: PolicyContext = {
            eventSource: 'NODE_EVENT',
            lifecycleStages: ['RUNTIME'],
        };

        it('should only apply to NODE_EVENT with RUNTIME lifecycle stage', () => {
            policyEventSources.forEach((eventSource) => {
                expect(
                    validator.appliesTo({
                        eventSource,
                        lifecycleStages: ['RUNTIME'],
                    })
                ).toBe(eventSource === 'NODE_EVENT');
            });
        });

        it('should pass when File Operation is not present', () => {
            const section: ClientPolicySection = {
                sectionName: 'Test Section',
                policyGroups: [mockCriterionWithName('Some Other Criterion')],
            };
            expect(validator.validate(section, context)).toBeUndefined();
        });

        it('should fail when File Operation is present but Actual Path is missing', () => {
            const section: ClientPolicySection = {
                sectionName: 'Test Section',
                policyGroups: [mockCriterionWithName('File Operation', [{ value: 'CREATE' }])],
            };
            const error = validator.validate(section, context);
            expect(error).toBeDefined();
        });

        it('should pass when File Operation and Actual Path both present with values', () => {
            const section: ClientPolicySection = {
                sectionName: 'Test Section',
                policyGroups: [
                    mockCriterionWithName('File Operation', [{ value: 'CREATE' }]),
                    mockCriterionWithName('Actual Path', [{ value: '/etc/passwd' }]),
                ],
            };
            expect(validator.validate(section, context)).toBeUndefined();
        });
    });
});

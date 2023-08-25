import { WizardPolicyStep4, initialExcludedDeployment, initialScope } from '../policies.utils';
import { validationSchemaStep4 } from './policyValidationSchemas';

// const options = { strict: true };

describe('Step 4', () => {
    it('passes if all properties have empty arrays', () => {
        const value: WizardPolicyStep4 = {
            scope: [],
            excludedDeploymentScopes: [],
            excludedImageNames: [],
        };
        expect(validationSchemaStep4.validateSync(value)).toEqual(value);
    });

    describe('scope', () => {
        it('throws if all properties have initial values', () => {
            const value: WizardPolicyStep4 = {
                scope: [initialScope],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });

        it('passes if cluster has non-empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        cluster: 'non-empty',
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if cluster has non-empty trimmed string', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        cluster: ' non-empty ',
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toBeDefined(); // returned value has trimmed string
        });

        it('passes if namespace has non-empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        namespace: 'non-empty',
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if namespace has non-empty trimmed string', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        namespace: ' non-empty ',
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toBeDefined(); // returned value has trimmed string
        });

        it('throws if key and value have undefined values', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: undefined,
                            value: undefined,
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });

        it('throws if key and value have empty strings', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: '',
                            value: '',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });

        it('throws if key and value have empty trimmed strings', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: ' ',
                            value: ' ',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });

        it('passes if key has non-empty string and value is absent', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: 'non-empty',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if key has non-empty string and value has undefined value', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: 'non-empty',
                            value: undefined,
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if key has non-empty string and value has empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: 'non-empty',
                            value: '',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if value has non-empty string and key is absent', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            value: 'non-empty',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if value has non-empty string and key has undefined value', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: undefined,
                            value: 'non-empty',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if value has non-empty string and key has empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: '',
                            value: 'non-empty',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if key and value have non-empty strings', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        ...initialScope,
                        label: {
                            key: 'non-empty',
                            value: 'non-empty',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if first scope has non-empty strings', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        cluster: 'non-empty',
                        namespace: 'non-empty',
                        label: {
                            key: 'non-empty',
                            value: 'non-empty',
                        },
                    },
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('throws if first scope has non-empty strings and second scope has initial values', () => {
            const value: WizardPolicyStep4 = {
                scope: [
                    {
                        cluster: 'non-empty',
                        namespace: 'non-empty',
                        label: {
                            key: 'non-empty',
                            value: 'non-empty',
                        },
                    },
                    initialScope,
                ],
                excludedDeploymentScopes: [],
                excludedImageNames: [],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });
    });

    describe('excludedDeploymentScopes', () => {
        it('throws if all properties have initial values', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [initialExcludedDeployment],
                excludedImageNames: [],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });

        it('passes if name has non-empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [
                    {
                        ...initialExcludedDeployment,
                        name: 'non-empty',
                    },
                ],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if cluster has non-empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [
                    {
                        ...initialExcludedDeployment,
                        scope: {
                            ...initialScope,
                            cluster: 'non-empty',
                        },
                    },
                ],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if namespace has non-empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [
                    {
                        ...initialExcludedDeployment,
                        scope: {
                            ...initialScope,
                            namespace: 'non-empty',
                        },
                    },
                ],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if key has non-empty string and value is absent', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [
                    {
                        ...initialExcludedDeployment,
                        scope: {
                            ...initialScope,
                            label: {
                                key: 'non-empty',
                            },
                        },
                    },
                ],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if value has non-empty string and key is absent', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [
                    {
                        ...initialExcludedDeployment,
                        scope: {
                            ...initialScope,
                            label: {
                                value: 'non-empty',
                            },
                        },
                    },
                ],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('passes if first excluded deployment has non-empty strings', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [
                    {
                        name: 'non-empty',
                        scope: {
                            cluster: 'non-empty',
                            namespace: 'non-empty',
                            label: {
                                key: 'non-empty',
                                value: 'non-empty',
                            },
                        },
                    },
                ],
                excludedImageNames: [],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('throws if first excluded deployment has non-empty strings and second excluded deployment has initial values', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [
                    {
                        name: 'non-empty',
                        scope: {
                            cluster: 'non-empty',
                            namespace: 'non-empty',
                            label: {
                                key: 'non-empty',
                                value: 'non-empty',
                            },
                        },
                    },
                    initialExcludedDeployment,
                ],
                excludedImageNames: [],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });
    });

    describe('excludedImageNames', () => {
        it('passes if first name is non-empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [],
                excludedImageNames: ['non-empty'],
            };
            expect(validationSchemaStep4.validateSync(value)).toEqual(value);
        });

        it('throws if first name is non-empty string but second string is empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [],
                excludedImageNames: ['non-empty', ''],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });

        it('throws if first name is empty string', () => {
            const value: WizardPolicyStep4 = {
                scope: [],
                excludedDeploymentScopes: [],
                excludedImageNames: [''],
            };
            expect(() => {
                validationSchemaStep4.validateSync(value);
            }).toThrow();
        });
    });
});

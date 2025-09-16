import uniqBy from 'lodash/uniqBy';

import { auditLogDescriptor, policyCriteriaDescriptors } from './policyCriteriaDescriptors';

// Enforce consistency of whicheverName properties in policy criteria descriptors.

// Add allowed items if unit tests fail for new rules that follow the rules (pardon pun).

// Items that are allowed independent of context.
const allowListForItems = [
    'API',
    'CPU',
    'CVE',
    'CVSS',
    'Dockerfile',
    'IP',
    'IPC',
    'Kubernetes',
    'MUST',
    'NOT',
    'OS',
    'PID',
    'RBAC',
    'UID',
    'USER',
];

// Items that are allowed only in the content of an entire string.
const allowListForNames = [
    'Common Vulnerability Scoring System (CVSS) score',
    'Common Vulnerability Scoring System (CVSS) score from National Vulnerability Database (NVD)',
    'Volume type (e.g. secret, configMap, hostPath) is',
    'Volume destination (mountPath) path is',
];

function isInitialUpperCase(item: string) {
    return /^[A-Z]/.test(item);
}

function isLowerCase(item: string) {
    return item === item.toLowerCase();
}

function hasSentenceCase(otherName: string) {
    return (
        allowListForNames.includes(otherName) ||
        otherName
            .split(' ')
            .every((item, i) =>
                i === 0
                    ? isInitialUpperCase(item)
                    : allowListForItems.includes(item) || isLowerCase(item)
            )
    );
}

describe('policyCriteriaDescriptors', () => {
    [...auditLogDescriptor, ...policyCriteriaDescriptors].forEach((descriptor) => {
        const { longName, name, shortName, type } = descriptor;

        describe(`descriptor of "${name}"`, () => {
            if (typeof longName === 'string') {
                test(`longName "${longName}" should not equal shortName`, () => {
                    expect(longName !== shortName).toEqual(true);
                });

                test(`longName "${longName}" should have sentence case`, () => {
                    expect(hasSentenceCase(longName)).toEqual(true);
                });
            }

            if ('negatedName' in descriptor && typeof descriptor.negatedName === 'string') {
                const { negatedName } = descriptor;

                test(`negatedName "${negatedName}" should not equal longName`, () => {
                    expect(negatedName !== longName).toEqual(true);
                });

                test(`negatedName "${negatedName}" should not equal shortName`, () => {
                    expect(negatedName !== shortName).toEqual(true);
                });

                test(`negatedName "${negatedName}" should have sentence case`, () => {
                    expect(hasSentenceCase(negatedName)).toEqual(true);
                });
            }

            test(`shortName "${shortName}" should have sentence case`, () => {
                expect(hasSentenceCase(shortName)).toEqual(true);
            });

            switch (type) {
                case 'group': {
                    const { subComponents } = descriptor;

                    subComponents.forEach((subComponent) => {
                        if (subComponent.type === 'select') {
                            const { options, subpath } = subComponent;

                            test(`group select "${subpath}" should have unique label properties`, () => {
                                const optionsUnique = uniqBy(options, ({ label }) => label);
                                expect(optionsUnique.length).toEqual(options.length);
                            });
                            test(`group select "${subpath}"  should have unique value properties`, () => {
                                const optionsUnique = uniqBy(options, ({ value }) => value);
                                expect(optionsUnique.length).toEqual(options.length);
                            });
                        }
                    });
                    break;
                }

                case 'multiselect':
                case 'select': {
                    const { options } = descriptor;

                    test(`${type} should have unique label properties`, () => {
                        const optionsUnique = uniqBy(options, ({ label }) => label);
                        expect(optionsUnique.length).toEqual(options.length);
                    });
                    test(`${type} should have unique value properties`, () => {
                        const optionsUnique = uniqBy(options, ({ value }) => value);
                        expect(optionsUnique.length).toEqual(options.length);
                    });
                    break;
                }

                case 'radioGroup': {
                    const { radioButtons } = descriptor;

                    test('radioGroup should have unique text properties', () => {
                        const radioButtonsUnique = uniqBy(radioButtons, ({ text }) => text);
                        expect(radioButtonsUnique.length).toEqual(radioButtons.length);
                    });
                    break;
                }

                case 'radioGroupString': {
                    const { radioButtons } = descriptor;

                    test('radioGroupString should have unique text properties', () => {
                        const radioButtonsUnique = uniqBy(radioButtons, ({ text }) => text);
                        expect(radioButtonsUnique.length).toEqual(radioButtons.length);
                    });
                    test('radioGroupString should have unique value properties', () => {
                        const radioButtonsUnique = uniqBy(radioButtons, ({ value }) => value);
                        expect(radioButtonsUnique.length).toEqual(radioButtons.length);
                    });
                    break;
                }

                default:
                    break;
            }
        });
    });
});

import { policyCriteriaDescriptors } from './policyCriteriaDescriptors';

// Enforce consistency of whicheverName properties in policy criteria descriptors.

// Add allowed items if unit tests fail for new rules that follow the rules (pardon pun).

// Items that are allowed independent of context.
const allowListForItems = [
    'CPU',
    'CVE',
    'Dockerfile',
    "doesn't",
    "don't",
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
const allowListForNames = ['Common Vulnerability Scoring System (CVSS) score'];

function isInitialUpperCase(item: string) {
    return /^[A-Z]/.test(item);
}

function isLowerCase(item: string) {
    return /^[a-z]+$/.test(item);
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
    policyCriteriaDescriptors.forEach((descriptor) => {
        const { longName, name, shortName } = descriptor;

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

            test(` shortName "${shortName}" should have sentence case`, () => {
                expect(hasSentenceCase(shortName)).toEqual(true);
            });
        });
    });
});

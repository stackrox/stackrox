/*
 * Messages of uncaught exceptions that are benign in test execution.
 *
 * ResizeObserver: Chrome fires it multiple times, which Cypress reports as an error.
 *   See https://github.com/cypress-io/cypress/issues/8418#issuecomment-992564877
 * MobX: the patternfly topology extension uses a 5.x version of Mobx, and the redoc library needs a 6.x version.
 *   TODO: remove this catch for multiple MobX versions after the in-product documentation is removed
 * getLanguageId: PatternFly code editor, related to our adding the YAML language module to it.
 */
const benignMessages = [
    'ResizeObserver loop completed',
    'ResizeObserver loop limit exceeded',
    'There are multiple, different versions of MobX active',
    'model.getLanguageId is not a function',
    "Uncaught SyntaxError: Unexpected token '<'",
];

/**
 * Whether an uncaught exception is known to be benign, so tests should not fail because of it.
 *
 * @param {Error} err
 * @returns {boolean}
 */
export function isBenignUncaughtException(err) {
    return benignMessages.some((message) => err.message.includes(message));
}

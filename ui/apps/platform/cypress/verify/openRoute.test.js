/*
 * Opens one route in the running app and records evidence of what happened.
 *
 * Run it through `scripts/verify.sh open <route>`, which sets these env vars:
 * - VERIFY_ROUTE: the route to open, for example /main/violations
 * - VERIFY_EVIDENCE_DIR: absolute path of the directory for report.json
 *
 * The test fails on failed API requests, uncaught exceptions, no API requests at all,
 * a "Cannot find the page" result, or a redirect to login. Console errors and accessibility
 * violations are recorded in the report as warnings, because some pages already have them.
 *
 * It uses cy.visit instead of the visit helper, because the helper fails as soon as a
 * page-level request is missing, before any evidence is written.
 */
import withAuth from '../helpers/basicAuth';
import { isBenignUncaughtException } from '../helpers/uncaughtExceptions';

// Same prefixes that the dev server proxies to Central, see src/setupProxy.js
const apiPathnamePattern = /^\/(v1|v2|api)\//;
const settleTimeoutMs = 30000;
const settleQuietMs = 1000;
const settlePollMs = 250;

/**
 * Waits until the page has made API requests and none has been in flight for a moment,
 * or the timeout passes. Resolves either way, so evidence is still written when the app
 * does not boot or a request hangs.
 *
 * @param {{ startedCount: number, pendingCount: number, lastActivityAt: number }} network
 * @returns {Cypress.Chainable<null>}
 */
function waitForNetworkToSettle(network) {
    const startedAt = Date.now();

    function check() {
        const now = Date.now();
        const isQuiet =
            network.startedCount !== 0 &&
            network.pendingCount === 0 &&
            now - network.lastActivityAt >= settleQuietMs;
        if (isQuiet || now - startedAt > settleTimeoutMs) {
            return cy.wrap(null, { log: false });
        }
        return cy.wait(settlePollMs, { log: false }).then(check);
    }

    return check();
}

/**
 * Returns the problems that make the verification fail.
 *
 * @param {Record<string, unknown>} evidence
 * @returns {string[]}
 */
function getFailures(evidence) {
    const failures = [];
    if (evidence.failedRequests.length !== 0) {
        failures.push(`${evidence.failedRequests.length} failed API request(s)`);
    }
    if (evidence.uncaughtExceptions.length !== 0) {
        failures.push(`${evidence.uncaughtExceptions.length} uncaught exception(s)`);
    }
    if (evidence.apiRequestCount === 0) {
        failures.push('the page made no API requests, so the app probably did not boot');
    }
    if (evidence.pendingRequestsAfterTimeout !== 0) {
        failures.push(`${evidence.pendingRequestsAfterTimeout} request(s) still pending`);
    }
    if (evidence.notFound) {
        failures.push('page rendered "Cannot find the page"');
    }
    if (evidence.redirectedToLogin) {
        failures.push('redirected to login, so the auth token is missing or invalid');
    }
    return failures;
}

describe('Verify: open route', () => {
    withAuth();

    it('should render the route without runtime errors', () => {
        cy.env(['VERIFY_ROUTE', 'VERIFY_EVIDENCE_DIR']).then(
            ({ VERIFY_ROUTE, VERIFY_EVIDENCE_DIR }) => {
                expect(VERIFY_ROUTE, 'VERIFY_ROUTE env var').to.be.a('string').and.not.be.empty;
                expect(VERIFY_EVIDENCE_DIR, 'VERIFY_EVIDENCE_DIR env var').to.be.a('string').and.not
                    .be.empty;

                const evidence = {
                    route: VERIFY_ROUTE,
                    finalUrl: '',
                    heading: '',
                    failures: [],
                    failedRequests: [],
                    uncaughtExceptions: [],
                    apiRequestCount: 0,
                    pendingRequestsAfterTimeout: 0,
                    notFound: false,
                    redirectedToLogin: false,
                    warnings: { consoleErrors: [], a11yViolations: [] },
                };
                const network = { startedCount: 0, pendingCount: 0, lastActivityAt: 0 };

                cy.on('uncaught:exception', (err) => {
                    if (!isBenignUncaughtException(err)) {
                        evidence.uncaughtExceptions.push(err.message);
                    }
                    // Keep going so the rest of the evidence is captured.
                    return false;
                });

                cy.on('window:before:load', (win) => {
                    const originalConsoleError = win.console.error;
                    // eslint-disable-next-line no-param-reassign
                    win.console.error = (...args) => {
                        evidence.warnings.consoleErrors.push(args.map(String).join(' '));
                        originalConsoleError.apply(win.console, args);
                    };
                });

                cy.intercept({ pathname: apiPathnamePattern }, (req) => {
                    network.startedCount += 1;
                    network.pendingCount += 1;
                    network.lastActivityAt = Date.now();
                    req.on('after:response', (res) => {
                        network.pendingCount -= 1;
                        network.lastActivityAt = Date.now();
                        if (res.statusCode >= 400) {
                            evidence.failedRequests.push({
                                method: req.method,
                                url: req.url,
                                status: res.statusCode,
                            });
                        }
                    });
                });

                cy.visit(VERIFY_ROUTE);

                waitForNetworkToSettle(network).then(() => {
                    evidence.apiRequestCount = network.startedCount;
                    evidence.pendingRequestsAfterTimeout = network.pendingCount;
                });

                cy.location().then((location) => {
                    evidence.finalUrl = `${location.pathname}${location.search}`;
                    evidence.redirectedToLogin = location.pathname.startsWith('/login');
                });

                cy.get('body').then(($body) => {
                    evidence.notFound =
                        $body.find('h1:contains("Cannot find the page")').length !== 0;
                    evidence.heading = $body.find('h1').first().text().trim();
                });

                cy.screenshot('page', { capture: 'fullPage' });

                cy.window().then((win) => {
                    if (!win.axe) {
                        cy.injectAxe({ axeCorePath: Cypress.env('AXE_CORE_PATH') });
                    }
                });
                const skipFailures = true;
                cy.checkA11y(
                    null,
                    null,
                    (violations) => {
                        evidence.warnings.a11yViolations = violations.map((violation) => ({
                            id: violation.id,
                            impact: violation.impact,
                            help: violation.help,
                            nodeCount: violation.nodes.length,
                        }));
                    },
                    skipFailures
                );

                cy.then(() => {
                    evidence.failures = getFailures(evidence);
                    cy.writeFile(`${VERIFY_EVIDENCE_DIR}/report.json`, evidence);
                });

                cy.then(() => {
                    expect(evidence.failures, 'verification failures').to.be.empty;
                });
            }
        );
    });
});

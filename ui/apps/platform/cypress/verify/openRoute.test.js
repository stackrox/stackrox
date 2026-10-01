/*
 * Opens one route in the running app and records evidence of what happened.
 *
 * Run it through `scripts/verify.sh open <route>`, which sets these env vars:
 * - VERIFY_ROUTE: the route to open, for example /main/violations
 * - VERIFY_EVIDENCE_DIR: absolute path of the directory for report.json
 * - VERIFY_HIGHLIGHT (optional): jQuery selector for the changed elements. They are outlined
 *   in a second full-page screenshot, and the run fails if the selector matches nothing.
 *
 * The test fails on failed API requests, uncaught exceptions, no API requests at all,
 * a loading indicator that never goes away, a "Cannot find the page" result, or a redirect
 * to login. Console errors and accessibility violations are recorded in the report as
 * warnings, because some pages already have them.
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
// PatternFly spinners and skeletons, matched without the version prefix (pf-v6-c-spinner).
const loadingIndicatorSelector = '[class*="-c-spinner"], [class*="-c-skeleton"]';
const highlightClassName = 'verify-highlight';
const highlightColor = '#c9190b';

/**
 * Waits until the page has made API requests, no request of any kind has been in flight
 * for a moment, and no loading indicator is visible, or the timeout passes.
 *
 * All requests count, not only API calls: the dev server loads a route's code in chunks
 * after the first API calls finish, and the page only asks for its data once that code
 * runs. Counting only API calls let the check pass before the page requested its data.
 *
 * Resolves either way, so evidence is still written when the app does not boot or a
 * request hangs. Resolves with whether a loading indicator was still visible.
 *
 * @param {{ apiStartedCount: number, pendingCount: number, lastActivityAt: number }} network
 * @returns {Cypress.Chainable<boolean>}
 */
function waitForPageToSettle(network) {
    const startedAt = Date.now();

    function check() {
        return cy.get('body', { log: false }).then(($body) => {
            const now = Date.now();
            const isLoading = $body.find(loadingIndicatorSelector).length !== 0;
            const isQuiet =
                network.apiStartedCount !== 0 &&
                network.pendingCount === 0 &&
                now - network.lastActivityAt >= settleQuietMs;
            if ((isQuiet && !isLoading) || now - startedAt > settleTimeoutMs) {
                return cy.wrap(isLoading, { log: false });
            }
            return cy.wait(settlePollMs, { log: false }).then(check);
        });
    }

    return check();
}

/**
 * Outlines the matched elements and labels the first one with the selector, so a reader of
 * the screenshot sees what changed. The outline is drawn inside the element, so a container
 * that clips overflow does not cut it off, and it does not change layout.
 *
 * @param {JQuery<HTMLElement>} $elements
 * @param {string} selector
 */
function addHighlight($elements, selector) {
    const doc = $elements[0].ownerDocument;
    $elements.each((_index, element) => {
        element.classList.add(highlightClassName);
    });

    const style = doc.createElement('style');
    style.id = highlightClassName;
    style.textContent = `.${highlightClassName} { outline: 3px solid ${highlightColor} !important; outline-offset: -3px; }`;
    doc.head.appendChild(style);

    const rect = $elements[0].getBoundingClientRect();
    const scrollingElement = doc.scrollingElement ?? doc.documentElement;
    const label = doc.createElement('div');
    label.id = `${highlightClassName}-label`;
    label.textContent = selector;
    Object.assign(label.style, {
        position: 'absolute',
        left: `${rect.left + scrollingElement.scrollLeft}px`,
        zIndex: '2147483647',
        padding: '2px 6px',
        background: highlightColor,
        color: '#fff',
        font: '12px monospace',
        pointerEvents: 'none',
    });
    doc.body.appendChild(label);
    // Place it above the element once its rendered height is known.
    label.style.top = `${Math.max(rect.top + scrollingElement.scrollTop - label.offsetHeight - 4, 0)}px`;
}

/**
 * Removes what addHighlight added, so the accessibility check sees the page as users do.
 *
 * @param {Document} doc
 */
function removeHighlight(doc) {
    doc.querySelectorAll(`.${highlightClassName}`).forEach((element) => {
        element.classList.remove(highlightClassName);
    });
    doc.getElementById(highlightClassName)?.remove();
    doc.getElementById(`${highlightClassName}-label`)?.remove();
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
    if (evidence.stillLoading) {
        failures.push('a loading indicator was still visible after the timeout');
    }
    if (evidence.notFound) {
        failures.push('page rendered "Cannot find the page"');
    }
    if (evidence.redirectedToLogin) {
        failures.push('redirected to login, so the auth token is missing or invalid');
    }
    if (evidence.highlight && evidence.highlight.matchCount === 0) {
        failures.push(`highlight selector ${evidence.highlight.selector} matched nothing`);
    }
    return failures;
}

describe('Verify: open route', () => {
    withAuth();

    it('should render the route without runtime errors', () => {
        cy.env(['VERIFY_ROUTE', 'VERIFY_EVIDENCE_DIR', 'VERIFY_HIGHLIGHT']).then(
            ({ VERIFY_ROUTE, VERIFY_EVIDENCE_DIR, VERIFY_HIGHLIGHT }) => {
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
                    stillLoading: false,
                    notFound: false,
                    redirectedToLogin: false,
                    warnings: { consoleErrors: [], a11yViolations: [] },
                    highlight: VERIFY_HIGHLIGHT
                        ? { selector: VERIFY_HIGHLIGHT, matchCount: 0 }
                        : null,
                };
                const network = { apiStartedCount: 0, pendingCount: 0, lastActivityAt: 0 };

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

                // Every request counts toward settling; only API responses can fail the run.
                cy.intercept({ url: '**' }, (req) => {
                    const isApiRequest = apiPathnamePattern.test(new URL(req.url).pathname);
                    if (isApiRequest) {
                        network.apiStartedCount += 1;
                    }
                    network.pendingCount += 1;
                    network.lastActivityAt = Date.now();
                    req.on('after:response', (res) => {
                        network.pendingCount -= 1;
                        network.lastActivityAt = Date.now();
                        if (isApiRequest && res.statusCode >= 400) {
                            evidence.failedRequests.push({
                                method: req.method,
                                url: req.url,
                                status: res.statusCode,
                            });
                        }
                    });
                });

                cy.visit(VERIFY_ROUTE);

                waitForPageToSettle(network).then((isLoading) => {
                    evidence.apiRequestCount = network.apiStartedCount;
                    evidence.pendingRequestsAfterTimeout = network.pendingCount;
                    evidence.stillLoading = isLoading;
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

                if (evidence.highlight) {
                    cy.get('body').then(($body) => {
                        const $matches = $body.find(VERIFY_HIGHLIGHT);
                        evidence.highlight.matchCount = $matches.length;
                        if ($matches.length !== 0) {
                            addHighlight($matches, VERIFY_HIGHLIGHT);
                            cy.screenshot('page-highlighted', { capture: 'fullPage' });
                            cy.document().then(removeHighlight);
                        }
                    });
                }

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

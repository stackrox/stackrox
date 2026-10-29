// Evidence helpers for verify-ui proof specs. Imported by absolute path from specs that
// scripts/new-spec.sh generates and by templates/support.js; never by checked-in tests.

// What drive.sh reads after the run to gate the proof. assertions maps each test that ran
// to the number of assertions in its body. Assertions in hooks, such as withAuth(), do not count.
// errors lists uncaught exceptions and console.error calls from the app.
const checks = { assertions: {}, errors: [] };

// Uncaught exceptions that cypress/support/e2e.js already ignores as benign. Keep in sync.
const knownBenignUncaught =
    /ResizeObserver loop (completed|limit exceeded)|multiple, different versions of MobX|model\.getLanguageId is not a function|Uncaught SyntaxError: Unexpected token '<'/;

function evidenceDir() {
    return Cypress.expose('VERIFY_EVIDENCE_DIR');
}

function testTitle() {
    return Cypress.currentTest ? Cypress.currentTest.titlePath.join(' > ') : '(outside a test)';
}

function recordError(kind, message) {
    checks.errors.push({ kind, test: testTitle(), message: String(message) });
}

/**
 * Record proof checks for every spec drive.sh runs. Called once from templates/support.js,
 * so it covers checked-in specs too; specs do not call it.
 */
export function installEvidenceHooks() {
    Cypress.on('log:added', (attrs) => {
        // A log in the test body has hookId equal to its testId; hooks get their own id.
        if (attrs.name === 'assert' && attrs.hookId === attrs.testId && Cypress.currentTest) {
            checks.assertions[testTitle()] += 1;
        }
    });

    // Recorded, and drive.sh fails the proof on any that is not known benign. Returns nothing,
    // so the repo handler still decides whether Cypress fails the test.
    Cypress.on('uncaught:exception', (err) => {
        const kind = knownBenignUncaught.test(err.message) ? 'uncaught (known benign)' : 'uncaught';
        // Keep the app's message, without the explanation Cypress wraps around it.
        const message = err.message
            .replace(/^The following error originated from your application code[^>]*>\s*/, '')
            .replace(/\s*When Cypress detects uncaught errors[\s\S]*$/, '');
        recordError(kind, message);
    });

    // Recorded only. The app logs some errors on its own, so these do not fail the proof.
    Cypress.on('window:before:load', (win) => {
        const original = win.console.error.bind(win.console);
        win.console.error = (...args) => {
            recordError('console.error', args.map(String).join(' '));
            original(...args);
        };
    });

    beforeEach(() => {
        checks.assertions[testTitle()] = 0;
    });

    after(() => {
        cy.writeFile(`${evidenceDir()}/checks.json`, checks, { log: false });
    });
}

/**
 * Crop a full-window PNG to the app frame and scale it to the app viewport size.
 *
 * @param {string} data base64 PNG of the whole runner window
 * @param {DOMRect} rect app frame rect in runner CSS pixels
 * @param {number} dpr runner device pixel ratio
 * @param {number} width app viewport width
 * @param {number} height app viewport height
 * @returns {Promise<string>} base64 PNG of the app only
 */
function cropToApp(data, rect, dpr, width, height) {
    return new Promise((resolve, reject) => {
        const image = new Image();
        image.onload = () => {
            const canvas = document.createElement('canvas');
            canvas.width = width;
            canvas.height = height;
            canvas
                .getContext('2d')
                .drawImage(image, rect.x * dpr, rect.y * dpr, rect.width * dpr, rect.height * dpr, 0, 0, width, height);
            resolve(canvas.toDataURL('image/png').split(',')[1]);
        };
        image.onerror = () => reject(new Error('verify-ui: could not decode window screenshot'));
        image.src = `data:image/png;base64,${data}`;
    });
}

/**
 * Screenshot the app frame through the DevTools protocol. Unlike cy.screenshot, this does not
 * hide the command log or rescale the app. The capture has no clip or scale because Chrome
 * applies those by briefly resizing the page, which the run video records as a flash; the
 * crop happens on a canvas instead.
 *
 * @param {string} label file name without extension
 */
function screenshotApp(label) {
    cy.window({ log: false }).then((win) => {
        // Scroll to the top so headings are in frame; assertions already ran.
        win.scrollTo(0, 0);
        win.document.querySelectorAll('*').forEach((el) => {
            if (el.scrollTop > 0) {
                el.scrollTop = 0;
            }
        });
    });
    cy.window({ log: false }).then((win) => {
        // The spec runs in a sibling frame; find the app frame from the runner window.
        const rect = window.top.document.querySelector('iframe.aut-iframe').getBoundingClientRect();
        const dpr = window.top.devicePixelRatio;
        return Cypress.automation('remote:debugger:protocol', {
            command: 'Page.captureScreenshot',
            params: { format: 'png' },
        })
            .then(({ data }) => cropToApp(data, rect, dpr, win.innerWidth, win.innerHeight))
            .then((png) => {
                const file = `${evidenceDir()}/screenshots/${Cypress.spec.name}/${label}.png`;
                cy.writeFile(file, png, 'base64', { log: false });
                cy.log(`screenshot ${label}`);
            });
    });
}

/**
 * Save a screenshot plus a JSON outline of what the user sees: URL, document title,
 * headings, alerts, and table row count. Use after each action whose result is the proof.
 *
 * @param {string} label short kebab-case step name, e.g. '02-after-filter'
 */
export function captureState(label) {
    screenshotApp(label);
    cy.document().then((doc) => {
        const texts = (selector) =>
            Array.from(doc.querySelectorAll(selector))
                .map((element) => element.textContent.trim().replace(/\s+/g, ' '))
                .filter(Boolean);
        cy.location().then((location) => {
            cy.writeFile(`${evidenceDir()}/${label}.state.json`, {
                url: location.pathname + location.search,
                title: doc.title,
                headings: texts('h1, h2, h3'),
                alerts: texts('[role="alert"], .pf-v6-c-alert__title'),
                tableRows: doc.querySelectorAll('tbody tr').length,
            });
        });
    });
}

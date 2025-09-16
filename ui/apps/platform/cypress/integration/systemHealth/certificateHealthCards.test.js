import withAuth from '../../helpers/basicAuth';
import {
    credentialForCentralExpiryAlias,
    credentialForCentralDbExpiryAlias,
    credentialForScannerExpiryAlias,
    setClock,
    visitSystemHealth,
} from '../../helpers/systemHealth';

const statusSelectorCentral = 'div:has(.pf-v5-c-card__header:contains("Central certificate"))';
const statusSelectorCentralDb =
    'div:has(.pf-v5-c-card__header:contains("Central Database certificate"))';
const statusSelectorScanner =
    'div:has(.pf-v5-c-card__header:contains("StackRox Scanner certificate"))';

describe('System Health Certificate Health Cards', () => {
    withAuth();

    describe('Central certificate', () => {
        it('should have widget and up to date text', () => {
            const currentDatetime = new Date('2025-05-21T02:04:59.377369440Z'); // about 355 days until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForCentralExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorCentral}:contains("expires in 12 months")`);
        });

        it('should have widget with warning text for less than 2 weeks', () => {
            const currentDatetime = new Date('2026-05-07T02:04:59.377369440Z'); // about 13 days until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForCentralExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorCentral}:contains("expires in 13 days")`);
        });

        it('should have widget with error text for less than 24 hours', () => {
            const currentDatetime = new Date('2026-05-19T02:05:59.377369440Z'); // about 23 hours, 59 minutes until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForCentralExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorCentral}:contains("expires in 1 day")`);
        });
    });

    describe('Central Database certificate', () => {
        it('should have widget and up to date text', () => {
            const currentDatetime = new Date('2025-05-21T02:04:59.377369440Z'); // about 355 days until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForCentralDbExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorCentralDb}:contains("expires in 12 months")`);
        });

        it('should have widget with warning text for less than 2 weeks', () => {
            const currentDatetime = new Date('2026-05-07T02:04:59.377369440Z'); // about 13 days until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForCentralDbExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorCentralDb}:contains("expires in 13 days")`);
        });

        it('should have widget with error text for less than 24 hours', () => {
            const currentDatetime = new Date('2026-05-19T02:05:59.377369440Z'); // about 23 hours, 59 minutes until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForCentralDbExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorCentralDb}:contains("expires in 1 day")`);
        });
    });

    describe('StackRox Scanner certificate', () => {
        it('should have widget and up to date text', () => {
            const currentDatetime = new Date('2025-05-21T02:04:59.377369440Z'); // about 355 days until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForScannerExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorScanner}:contains("expires in 12 months")`);
        });

        it('should have widget with warning text for less than 2 weeks', () => {
            const currentDatetime = new Date('2026-05-07T02:04:59.377369440Z'); // about 13 days until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForScannerExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorScanner}:contains("expires in 13 days")`);
        });

        it('should have widget with error text for less than 24 hours', () => {
            const currentDatetime = new Date('2026-05-19T02:05:59.377369440Z'); // about 23 hours, 59 minutes until expiry
            const expiry = '2026-05-20T03:04:59.377369440Z';

            const staticResponseMap = {
                [credentialForScannerExpiryAlias]: { body: { expiry } },
            };

            setClock(currentDatetime); // call before visit
            visitSystemHealth(staticResponseMap);

            cy.get(`${statusSelectorScanner}:contains("expires in 1 day")`);
        });
    });
});

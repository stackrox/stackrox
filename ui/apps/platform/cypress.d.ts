import { mount } from 'cypress/react';

declare global {
    namespace Cypress {
        interface Chainable {
            mount: typeof mount;
            // This is a workaround for the Cypress issue where it doesn't recognize the response
            // type when using an '@' alias. This will result in a return type of 'any' instead.
            //
            // See issue: https://github.com/cypress-io/cypress/issues/24823
            get<T extends any = any>(alias: `@${string}`): Chainable<T>;
        }
    }
}

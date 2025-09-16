import path from 'path';

export function readFileFromDownloads(filename: string) {
    return cy.readFile(path.join(Cypress.config('downloadsFolder'), filename));
}

export function readFileFromDownloads(filename: string) {
    return cy
        .task<string>('joinPaths', [Cypress.config('downloadsFolder'), filename])
        .then((path) => {
            return cy.readFile(path);
        });
}

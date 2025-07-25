import { selectors as topNavSelectors } from '../constants/TopNavigation';
import { visitMainDashboard } from './main';
import { visit, visitWithStaticResponseForAuthStatus } from './visit';

const basePath = '/main/user';

const title = 'User Profile';

export function visitUserProfile() {
    visit(basePath);

    cy.get(`h1:contains("${title}")`);
}

export function visitUserProfileFromTopNav() {
    visitMainDashboard();

    cy.get(topNavSelectors.menuButton).click();
    cy.get(topNavSelectors.menuList.myProfileButton).click();

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);
}

export function visitUserProfileWithStaticResponseForAuthStatus(staticResponseForAuthStatus) {
    visitWithStaticResponseForAuthStatus(basePath, staticResponseForAuthStatus);

    cy.get(`h1:contains("${title}")`);
}

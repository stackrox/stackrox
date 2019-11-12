import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';

const hasExpectedHeaderColumns = colNames => {
    colNames.forEach(col => {
        cy.get(`${selectors.tableColumn}:contains('${col}')`);
    });
};

const hasExpectedLinks = colLinks => {
    colLinks.forEach(col => {
        cy.get(`${selectors.tableColumnLinks}:contains('${col.toLowerCase()}')`);
    });
};

const hasExpectedCVELinks = colCVELinks => {
    colCVELinks.forEach(col => {
        cy.get(`${selectors.tableCVEColumnLinks}:contains('${col}')`);
    });
};

describe('Entities list Page', () => {
    withAuth();
    it('should display all the columns and links expected in clusters list page', () => {
        cy.visit(url.list.clusters);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'K8S version',
            'Namespaces',
            'Deployments',
            'Policies',
            'Policy status',
            'Latest violation',
            'Risk Priority'
        ]);
        hasExpectedLinks(['Namespace', 'Deployment']);
        hasExpectedCVELinks(['CVE', 'Fixable']);
    });

    it('should display all the columns and links expected in namespaces list page', () => {
        cy.visit(url.list.namespaces);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'Images',
            'Namespace',
            'Deployments',
            'Policies',
            'Policy status',
            'Latest violation',
            'Risk Priority'
        ]);
        hasExpectedLinks(['image', 'deployment']);
        hasExpectedCVELinks(['CVE', 'Fixable']);
    });

    it('should display all the columns and links expected in deployments list page', () => {
        cy.visit(url.list.deployments);
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'Images',
            'Namespace',
            'Deployment',
            'Policies',
            'Policy Status',
            'Latest violation',
            'Risk Priority'
        ]);
        hasExpectedLinks(['image']);
        hasExpectedCVELinks(['CVE', 'Fixable']);
    });

    it('should display all the columns and links expected in images list page', () => {
        cy.visit(url.list.images);
        hasExpectedHeaderColumns([
            'Image',
            'CVEs',
            'Top CVSS',
            'Created',
            'Scan time',
            'Image Status',
            'Deployments',
            'Components',
            'Risk Priority'
        ]);
        hasExpectedLinks(['deployment', 'component']);
        hasExpectedCVELinks(['CVE', 'Fixable']);
    });

    it('should display all the columns expected in components list page', () => {
        cy.visit(url.list.components);
        hasExpectedHeaderColumns([
            'Component',
            'CVEs',
            'Top CVSS',
            'Images',
            'Deployments',
            'Risk Priority'
        ]);
        hasExpectedLinks(['deployment', 'image']);
        hasExpectedCVELinks(['CVE', 'Fixable']);
    });

    it('should display all the columns and links  expected in cves list page', () => {
        cy.visit(url.list.cves);
        hasExpectedHeaderColumns([
            'CVE',
            'Fixable',
            'CVSS score',
            'Env. Impact',
            'Impact score',
            'Scanned',
            'Published',
            'Deployments'
        ]);
        hasExpectedLinks(['image', 'deployment', 'component']);
    });
});

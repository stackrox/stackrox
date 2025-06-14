import React, { useState } from 'react';
import SnoozedCveToggleButton from './SnoozedCveToggleButton';

function Wrapper({ startingSearchFilter = {}, snoozedCveCount }) {
    const [searchFilter, setSearchFilter] = useState(startingSearchFilter);

    return (
        <>
            <div>
                <h1>Filters</h1>
                {Object.entries(searchFilter).map(([key, values]) => (
                    <div key={key}>
                        {key}:{values.join(',')}
                    </div>
                ))}
            </div>
            <SnoozedCveToggleButton
                searchFilter={searchFilter}
                setSearchFilter={setSearchFilter}
                snoozedCveCount={snoozedCveCount}
            />
        </>
    );
}

const snoozedFilterSelector =
    'div:has(h1:contains("Filters")) div:contains("CVE Snoozed:true") div';

const severityFilterSelector =
    'div:has(h1:contains("Filters")) div:contains("Severity:Critical,Important") div';

describe(Cypress.spec.relative, () => {
    it('should manage the toggling of the snoozed CVE filter', () => {
        cy.mount(<Wrapper />);

        // Default is off
        cy.get(snoozedFilterSelector).should('not.exist');

        // Toggle on
        cy.findByText('Show snoozed CVEs').click();
        cy.get(snoozedFilterSelector).should('exist');

        // Toggle off
        cy.findByText('Show observed CVEs').click();
        cy.get(snoozedFilterSelector).should('not.exist');
    });

    it('should not change the state of existing filters when toggling the snoozed CVE filter', () => {
        cy.mount(<Wrapper startingSearchFilter={{ Severity: ['Critical', 'Important'] }} />);

        // Default is off
        cy.get(snoozedFilterSelector).should('not.exist');
        cy.get(severityFilterSelector).should('exist');

        // Toggle on
        cy.findByText('Show snoozed CVEs').click();
        cy.get(snoozedFilterSelector).should('exist');
        cy.get(severityFilterSelector).should('exist');

        // Toggle off
        cy.findByText('Show observed CVEs').click();
        cy.get(snoozedFilterSelector).should('not.exist');
        cy.get(severityFilterSelector).should('exist');
    });

    it('should correctly display the current snoozed CVE count', () => {
        // Badge should not show when no count is provided
        cy.mount(<Wrapper snoozedCveCount={undefined} />);
        cy.get('button .pf-v5-c-badge').should('not.exist');

        // Badge should not show when count is 0
        cy.mount(<Wrapper snoozedCveCount={0} />);
        cy.get('button .pf-v5-c-badge').should('not.exist');

        // Badge should show when count is > 0
        cy.mount(<Wrapper snoozedCveCount={1} />);
        cy.get('button .pf-v5-c-badge').contains('1');

        // Badge should not show when viewing snoozed CVEs
        cy.findByText('Show snoozed CVEs').click();
        cy.get('button .pf-v5-c-badge').should('not.exist');
    });
});

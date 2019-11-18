import React from 'react';

import renderWithRouter from 'test-utils/renderWithRouter';
import VulnMgmtCveOverview from './VulnMgmtCveOverview';

describe('VulnMgmtComponentCveOverview', () => {
    it('renders an external link to more info about the CVE', () => {
        // arrange
        const mockExternalLink = 'https://security-tracker.debian.org/tracker/CVE-2019-9923';
        const data = {
            cve: 'CVE-2019-9923',
            envImpact: 0.375,
            cvss: 7.5,
            scoreVersion: 'V3',
            link: mockExternalLink,
            vectors: {
                impactScore: 3.5999999046325684,
                exploitabilityScore: 3.9000000953674316,
                vector: 'CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H'
            },
            publishedOn: '2019-03-22T08:29:00Z',
            lastModified: '2019-04-24T19:02:00Z',
            summary:
                'pax_decode_header in sparse.c in GNU Tar before 1.32 had a NULL pointer dereference when parsing certain archives that have malformed extended headers.',
            fixedByVersion: '',
            isFixable: false
        };

        // act
        const { getByTestId } = renderWithRouter(<VulnMgmtCveOverview data={data} />, {
            route: `/cve/${data.cve}`
        });

        // assert
        const el = getByTestId('more-info-link');
        expect(el).toHaveTextContent('View Full CVE Description');
        expect(el).toHaveAttribute('href', mockExternalLink);
        expect(el).toHaveAttribute('target', '_blank');
        expect(el).toHaveAttribute('rel', 'noopener noreferrer nofollow');
    });
});

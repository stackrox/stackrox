/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React from 'react';
import ObservedCVEsTable, { ObservedCVERow } from './ObservedCVEsTable';

const rows = [
    {
        id: 'CVE-2014-232',
        cve: 'CVE-2014-232',
        isFixable: false,
        severity: 'MODERATE_VULNERABILITY_SEVERITY',
        cvssScore: '5.8',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'perl 5.24.1-3+deb9u5',
                fixedIn: 'struts-232',
            },
        ],
        discoveredAt: '3 days ago',
    },
    {
        id: 'CVE-2019-5953',
        cve: 'CVE-2019-5953',
        isFixable: true,
        severity: 'CRITICAL_VULNERABILITY_SEVERITY',
        cvssScore: '9.8',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
        ],
        discoveredAt: '2 days ago',
    },
    {
        id: 'CVE-2017-13090',
        cve: 'CVE-2017-13090',
        isFixable: true,
        severity: 'IMPORTANT_VULNERABILITY_SEVERITY',
        cvssScore: '8.8',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
        ],
        discoveredAt: '3 days ago',
    },
    {
        id: 'CVE-2016-7098',
        cve: 'CVE-2016-7098',
        isFixable: true,
        severity: 'IMPORTANT_VULNERABILITY_SEVERITY',
        cvssScore: '8.1',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
        ],
        discoveredAt: '3 days ago',
    },
    {
        id: 'CVE-2018-0494',
        cve: 'CVE-2018-0494',
        isFixable: true,
        severity: 'MODERATE_VULNERABILITY_SEVERITY',
        cvssScore: '6.5',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'perl 5.24.1-3+deb9u5',
                fixedIn: 'struts-232',
            },
        ],
        discoveredAt: '5 days ago',
    },
] as ObservedCVERow[];

function ObservedCVEs() {
    // @TODO: hook to GET Observed CVEs data goes here

    return <ObservedCVEsTable rows={rows} />;
}

export default ObservedCVEs;

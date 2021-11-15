/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React from 'react';
import FalsePositiveCVEsTable, { FalsePositiveCVERow } from './FalsePositiveCVEsTable';

const rows = [
    {
        id: 'CVE-2014-232',
        cve: 'CVE-2014-232',
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
        comments: [
            {
                id: '1',
                user: 'Trevor',
                message: "Update fix isn't ready",
                createdAt: '12/21/2020 | 4:24 AM',
            },
            {
                id: '2',
                user: 'Jacob',
                message: 'Get it done!',
                createdAt: '12/21/2020 | 4:50 AM',
            },
        ],
        applyTo: 'All image tags',
    },
    {
        id: 'CVE-2019-5953',
        cve: 'CVE-2019-5953',
        severity: 'CRITICAL_VULNERABILITY_SEVERITY',
        cvssScore: '9.8',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
        ],
        comments: [
            {
                id: '1',
                user: 'Trevor',
                message: "Update fix isn't ready",
                createdAt: '12/21/2020 | 4:24 AM',
            },
            {
                id: '2',
                user: 'Jacob',
                message: 'Get it done!',
                createdAt: '12/21/2020 | 4:50 AM',
            },
        ],
        applyTo: 'All image tags',
    },
    {
        id: 'CVE-2017-13090',
        cve: 'CVE-2017-13090',
        severity: 'IMPORTANT_VULNERABILITY_SEVERITY',
        cvssScore: '8.8',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
        ],
        comments: [
            {
                id: '1',
                user: 'Trevor',
                message: "Update fix isn't ready",
                createdAt: '12/21/2020 | 4:24 AM',
            },
            {
                id: '2',
                user: 'Jacob',
                message: 'Get it done!',
                createdAt: '12/21/2020 | 4:50 AM',
            },
        ],
        applyTo: 'All image tags',
    },
    {
        id: 'CVE-2016-7098',
        cve: 'CVE-2016-7098',
        severity: 'IMPORTANT_VULNERABILITY_SEVERITY',
        cvssScore: '8.1',
        components: [
            {
                id: 'b3BlbnNzbA:MS4xLjFkLTArZGViMTB1Mg',
                name: 'glibc 2.24-11+deb9u4',
                fixedIn: 'struts-232',
            },
        ],
        comments: [
            {
                id: '1',
                user: 'Trevor',
                message: "Update fix isn't ready",
                createdAt: '12/21/2020 | 4:24 AM',
            },
            {
                id: '2',
                user: 'Jacob',
                message: 'Get it done!',
                createdAt: '12/21/2020 | 4:50 AM',
            },
        ],
        applyTo: 'All image tags',
    },
] as FalsePositiveCVERow[];

function FalsePositiveCVEs() {
    // @TODO: hook to GET false positive CVEs data goes here

    return <FalsePositiveCVEsTable rows={rows} />;
}

export default FalsePositiveCVEs;

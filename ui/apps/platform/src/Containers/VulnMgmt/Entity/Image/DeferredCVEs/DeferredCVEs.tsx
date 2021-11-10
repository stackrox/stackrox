/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React from 'react';
import DeferredCVEsTable, { DeferredCVERow } from './DeferredCVEsTable';

const rows = [
    {
        id: 'CVE-2014-232',
        cve: 'CVE-2014-232',
        severity: 'MODERATE_VULNERABILITY_SEVERITY',
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
        expiresAt: 'in 2 weeks',
        applyTo: 'All image tags',
        approver: 'Jacob',
    },
    {
        id: 'CVE-2015-532',
        cve: 'CVE-2015-532',
        severity: 'MODERATE_VULNERABILITY_SEVERITY',
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
        expiresAt: 'in 2 weeks',
        applyTo: 'All image tags',
        approver: 'Jacob',
    },
    {
        id: 'CVE-2016-322',
        cve: 'CVE-2065-322',
        severity: 'MODERATE_VULNERABILITY_SEVERITY',
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
        expiresAt: 'in 2 weeks',
        applyTo: 'All image tags',
        approver: 'Jacob',
    },
] as DeferredCVERow[];

function DeferredCVEs() {
    // @TODO: hook to GET Deferred CVEs data goes here

    return <DeferredCVEsTable rows={rows} />;
}

export default DeferredCVEs;

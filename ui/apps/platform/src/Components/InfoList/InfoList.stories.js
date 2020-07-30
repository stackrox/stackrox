/* eslint-disable no-use-before-define */
import React from 'react';

import InfoList from './InfoList';

export default {
    title: 'InfoList',
    component: InfoList,
};

export const basicInfoList = () => {
    const data = ['CVE-2005-2541', 'CVE-2017-12424', 'CVE-2018-16402'];

    return <InfoList items={data} />;
};

export const withCustomItemRenderer = () => {
    const data = [
        {
            id: 'CVE-2017-14062',
            cve: 'CVE-2017-14062',
            summary:
                'Integer overflow in the decode_digit function in puny_decode.c in Libidn2 before 2.0.4 allows remote attackers to cause a denial of service or possibly have unspecified other impact.',
        },
        {
            id: 'CVE-2018-16402',
            cve: 'CVE-2018-16402',
            summary:
                'libelf/elf_end.c in elfutils 0.173 allows remote attackers to cause a denial of service (double free and application crash) or possibly have unspecified other impact because it tries to decompress twice.',
        },
        {
            id: 'CVE-2018-6485',
            cve: 'CVE-2018-6485',
            summary:
                'An integer overflow in the implementation of the posix_memalign in memalign functions in the GNU C Library (aka glibc or libc6) 2.26 and earlier could cause these functions to return a pointer to a heap area that is too small, potentially leading to heap corruption.',
        },
    ];
    function customRenderer(item) {
        return (
            <li key={item.id} className="flex items-center bg-tertiary-200 mb-2 p-2">
                <span className="min-w-48">{item.cve}</span>
                <span>{item.summary}</span>
            </li>
        );
    }

    return <InfoList items={data} renderItem={customRenderer} />;
};

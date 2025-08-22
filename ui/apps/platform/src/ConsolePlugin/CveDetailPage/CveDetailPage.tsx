import React from 'react';
import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';
import { useParams } from 'react-router-dom-v5-compat';

export function CveDetailPage() {
    const { cveId } = useParams();
    const [namespace] = useActiveNamespace();
    return (
        <>
            <div>CVE Detail Page</div>
            <div>Namespace: {namespace}</div>
            <div>CVE ID: {cveId}</div>
        </>
    );
}

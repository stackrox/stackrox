import React from 'react';
import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';
import { useParams } from 'react-router-dom-v5-compat';

export function ImageDetailPage() {
    const [namespace] = useActiveNamespace();
    const { imageId } = useParams();

    return (
        <>
            <div>Image Detail Page</div>
            <div>Namespace: {namespace}</div>
            <div>Image ID: {imageId}</div>
        </>
    );
}

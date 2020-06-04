import React from 'react';

import CveType from './CveType';

export default {
    title: 'CveType',
    component: CveType,
};

export const withDefaultStyle = () => (
    <div>
        <CveType type="IMAGE_CVE" />
        <h2 className="mt-8 mb-0">Text inherits its parent element properties</h2>
        <p>(example in a table cell)</p>
        <div className="rt-td w-1/10 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal">
            <CveType type="IMAGE_CVE" />
        </div>
    </div>
);

export const withDefaultStyleK8sType = () => (
    <div>
        <CveType type="K8S_CVE" />
        <h2 className="mt-8 mb-0">Text inherits its parent element properties</h2>
        <p>(example in a table cell)</p>
        <div className="rt-td w-1/10 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal">
            <CveType type="K8S_CVE" />
        </div>
    </div>
);

export const withDefaultStyleIstioType = () => (
    <div>
        <CveType type="ISTIO_CVE" />
        <h2 className="mt-8 mb-0">Text inherits its parent element properties</h2>
        <p>(example in a table cell)</p>
        <div className="rt-td w-1/10 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal">
            <CveType type="ISTIO_CVE" />
        </div>
    </div>
);

export const withCalloutStyle = () => <CveType context="callout" type="IMAGE_CVE" />;

export const withCalloutStyleK8sType = () => <CveType context="callout" type="K8S_CVE" />;

export const withCalloutStyleIstioType = () => <CveType context="callout" type="ISTIO_CVE" />;

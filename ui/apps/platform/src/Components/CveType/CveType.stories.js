import React from 'react';

import CveType from './CveType';

export default {
    title: 'CveType',
    component: CveType,
};

export const withDefaultStyle = () => (
    <div>
        <CveType types={['IMAGE_CVE']} />
        <h2 className="mt-8 mb-0">Text inherits its parent element properties</h2>
        <p>(example in a table cell)</p>
        <div className="rt-td w-1/10 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal">
            <CveType types={['IMAGE_CVE']} />
        </div>
    </div>
);

export const withDefaultStyleK8sType = () => (
    <div>
        <CveType types={['K8S_CVE']} />
        <h2 className="mt-8 mb-0">Text inherits its parent element properties</h2>
        <p>(example in a table cell)</p>
        <div className="rt-td w-1/10 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal">
            <CveType types={['K8S_CVE']} />
        </div>
    </div>
);

export const withDefaultStyleIstioType = () => (
    <div>
        <CveType types={['ISTIO_CVE']} />
        <h2 className="mt-8 mb-0">Text inherits its parent element properties</h2>
        <p>(example in a table cell)</p>
        <div className="rt-td w-1/10 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal">
            <CveType types={['ISTIO_CVE']} />
        </div>
    </div>
);

export const withCalloutStyle = () => <CveType context="callout" types={['IMAGE_CVE']} />;

export const withCalloutStyleK8sType = () => <CveType context="callout" types={['K8S_CVE']} />;

export const withCalloutStyleIstioType = () => <CveType context="callout" types={['ISTIO_CVE']} />;

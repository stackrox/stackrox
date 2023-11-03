/* eslint jest/expect-expect: ["error", { "assertFunctionNames": ["expectColumnsToContain", "expectColumnsNotToContain"] }] */

import { getColumnsByEntity, getColumnsByStandard } from './tableColumns';
import { resourceTypes, standardTypes } from './entityTypes';

function expectColumnsToContain(columns, shouldContain) {
    const accessors = columns.map((c) => c.accessor);
    expect(accessors).toEqual(expect.arrayContaining(shouldContain));
}

function expectColumnsNotToContain(columns, shouldNotContain) {
    const accessors = columns.map((c) => c.accessor);
    expect(accessors).toEqual(expect.not.arrayContaining(shouldNotContain));
}

const standards = [
    {
        id: 'CIS_Kubernetes_v1_5',
        name: 'CIS Kubernetes v1.5',
        scopes: ['CLUSTER', 'NODE'],
    },
    {
        id: 'HIPAA_164',
        name: 'HIPAA 164',
        scopes: ['CLUSTER', 'NAMESPACE', 'DEPLOYMENT'],
    },
    {
        id: 'NIST_800_190',
        name: 'NIST SP 800-190',
        scopes: ['CLUSTER', 'NAMESPACE', 'DEPLOYMENT', 'NODE'],
    },
    {
        id: 'NIST_SP_800_53_Rev_4',
        name: 'NIST SP 800-53',
        scopes: ['CLUSTER', 'NAMESPACE', 'DEPLOYMENT'],
    },
    {
        id: 'PCI_DSS_3_2',
        name: 'PCI DSS 3.2.1',
        scopes: ['CLUSTER', 'NAMESPACE', 'DEPLOYMENT'],
    },
];

describe('Get columns', () => {
    it('can get columns by entity with exclusion', () => {
        expectColumnsToContain(getColumnsByEntity(resourceTypes.CLUSTER, standards), [
            'id',
            standardTypes.NIST_800_190,
            standardTypes.NIST_SP_800_53_Rev_4,
        ]);

        expectColumnsToContain(getColumnsByEntity(resourceTypes.NODE, standards), [
            standardTypes.NIST_800_190,
        ]);

        expectColumnsNotToContain(getColumnsByEntity(resourceTypes.NODE, standards), [
            standardTypes.NIST_SP_800_53_Rev_4,
        ]);
    });

    it('can get columns by standard', () => {
        expectColumnsToContain(getColumnsByStandard(standardTypes.NIST_SP_800_53_Rev_4), [
            'id',
            'compliance',
            'control',
        ]);
    });
});

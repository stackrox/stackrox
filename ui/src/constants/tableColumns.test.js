import { getColumnsByEntity, getColumnsByStandard } from './tableColumns';
import { resourceTypes, standardTypes } from './entityTypes';

function expectColumnsToContain(columns, shouldContain) {
    const accessors = columns.map(c => c.accessor);
    expect(accessors).toEqual(expect.arrayContaining(shouldContain));
}

function expectColumnsNotToContain(columns, shouldNotContain) {
    const accessors = columns.map(c => c.accessor);
    expect(accessors).toEqual(expect.not.arrayContaining(shouldNotContain));
}

describe('Get columns', () => {
    it('can get columns by entity without exclusion', () => {
        expectColumnsToContain(getColumnsByEntity(resourceTypes.CLUSTER), [
            'id',
            standardTypes.NIST_800_190,
            standardTypes.NIST_SP_800_53_Rev_4
        ]);

        expectColumnsNotToContain(getColumnsByEntity(resourceTypes.NODE), [
            standardTypes.NIST_SP_800_53_Rev_4
        ]);
    });

    it('can get columns by entity with exclusion', () => {
        const columns = getColumnsByEntity(resourceTypes.CLUSTER, [
            standardTypes.NIST_SP_800_53_Rev_4
        ]);
        expectColumnsToContain(columns, ['id', standardTypes.NIST_800_190]);
        expectColumnsNotToContain(columns, [standardTypes.NIST_SP_800_53_Rev_4]);
    });

    it('can get columns by standard', () => {
        expectColumnsToContain(getColumnsByStandard(standardTypes.NIST_SP_800_53_Rev_4), [
            'id',
            'compliance',
            'control'
        ]);
    });
});

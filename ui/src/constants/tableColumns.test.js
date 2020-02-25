import { getColumnsByEntity, getColumnsByStandard } from './tableColumns';
import { resourceTypes, standardTypes } from './entityTypes';

function expectColumnsToContainAndNotContain(columns, shouldContain, shouldNotContain) {
    const accessors = columns.map(c => c.accessor);
    expect(accessors).toEqual(expect.arrayContaining(shouldContain));
    if (shouldNotContain) {
        expect(accessors).toEqual(expect.not.arrayContaining(shouldNotContain));
    }
}

describe('Get columns', () => {
    it('can get columns by entity without exclusion', async () => {
        expectColumnsToContainAndNotContain(getColumnsByEntity(resourceTypes.CLUSTER), [
            'id',
            standardTypes.NIST_800_190,
            standardTypes.NIST_SP_800_53
        ]);
    });

    it('can get columns by entity with exclusion', async () => {
        expectColumnsToContainAndNotContain(
            getColumnsByEntity(resourceTypes.CLUSTER, [standardTypes.NIST_SP_800_53]),
            ['id', standardTypes.NIST_800_190],
            [standardTypes.NIST_SP_800_53]
        );
    });

    it('can get columns by standard', async () => {
        expectColumnsToContainAndNotContain(getColumnsByStandard(standardTypes.NIST_SP_800_53), [
            'id',
            'compliance',
            'control'
        ]);
    });
});

import intersection from 'lodash/intersection';

import entityRelationships, {
    entityGroupMap,
    getVulnerabilityManagementEntityTypesByRelationship as getEntityTypesByRelationship,
} from './entityRelationships';

describe('entityRelationshipMap', () => {
    // Assume that entityGroupMap has all entity types which have relationships.
    const entityTypesRelationships = Object.keys(entityGroupMap).sort();

    /**
     * Emulate get*s function that is not exported
     * using is* callback function that is exported.
     * The returned array is sorted, because entityTypes is sorted.
     */
    const enumerateTypes = (entityType1, callback) =>
        entityTypesRelationships.filter((entityType2) => callback(entityType1, entityType2));

    const {
        getChildren,
        getContains,
        getMatches,
        getParents,
        isContainedInferred,
        isExtendedMatch,
        isPureMatch,
    } = entityRelationships;

    const receivedRelationships = {};
    entityTypesRelationships.forEach((entityType) => {
        const children = [...getChildren(entityType)].sort(); // copy before sort
        const parents = [...getParents(entityType)].sort();
        const matches = enumerateTypes(entityType, isPureMatch);
        const extendedMatches = enumerateTypes(entityType, isExtendedMatch);
        const expectedMatches = [...matches, ...extendedMatches].sort();
        const receivedMatches = [...getMatches(entityType)].sort();
        const receivedContains = [...getContains(entityType)].sort();
        const receivedInferred = enumerateTypes(entityType, isContainedInferred);

        // Numbers in keys for order to compare with entityRelationshipMap.
        // Spaces in keys for alignment to compare related relationships.
        receivedRelationships[entityType] = {
            '1-children': children.join(' '),
            '2-parents': parents.join(' '),
            '3-matches        ': matches.join(' '),
            '4-extendedMatches': extendedMatches.join(' '),
            '5-getMatches     ': receivedMatches.join(' '),
            '6-getContains        ': receivedContains.join(' '),
            '7-isContainedInferred': receivedInferred.join(' '),
        };

        // Template literal for jest/valid-title which forbids a variable.
        describe(`${entityType}`, () => {
            it('has relationships that are not reflexive', () => {
                expect(children).not.toContain(entityType);
                expect(parents).not.toContain(entityType);
                expect(expectedMatches).not.toContain(entityType);
                expect(receivedContains).not.toContain(entityType);
                expect(receivedInferred).not.toContain(entityType);
            });

            it(`has properties that are disjoint`, () => {
                expect(intersection(children, parents, matches, extendedMatches)).toEqual([]);
            });

            it(`has getMatches method that is union of matches and extendedMatches`, () => {
                expect(receivedMatches).toEqual(expectedMatches);
            });

            it(`has isContained method is union of children and isContainedInferred`, () => {
                const expectedContains = [...children, ...receivedInferred].sort();
                expect(receivedContains).toEqual(expectedContains);
            });
        });
    });

    // Snapshot shows the effect of changes to data or logic.
    it('has properties and derived properties', () => {
        expect(receivedRelationships).toMatchSnapshot(); // not filtered by use case
    });
});

it('gets Matches relationships for Cluster', () => {
    const matchesForCluster = getEntityTypesByRelationship('CLUSTER', 'MATCHES');
    expect(matchesForCluster).not.toContain('CONTROL');
});

it('gets Matches relationships for Image CVE', () => {
    // should also get extended matches
    const matchesForImageCVE = getEntityTypesByRelationship('IMAGE_CVE', 'MATCHES');
    expect(matchesForImageCVE).toContain('IMAGE');
    expect(matchesForImageCVE).toContain('IMAGE_COMPONENT');
    expect(matchesForImageCVE).toContain('DEPLOYMENT');
    expect(matchesForImageCVE).not.toContain('NAMESPACE');
});

it('gets direct Children relationships for Cluster', () => {
    const childrenOfCluster = getEntityTypesByRelationship('CLUSTER', 'CHILDREN');
    expect(childrenOfCluster).toContain('NODE');
    expect(childrenOfCluster).toContain('NAMESPACE');
    expect(childrenOfCluster).not.toContain('ROLE');
});

it('gets Contains relationships for Cluster', () => {
    const containsForCluster = getEntityTypesByRelationship('CLUSTER', 'CONTAINS');

    // should have direct child
    expect(containsForCluster).not.toContain('ROLE');
    expect(containsForCluster).toContain('NAMESPACE');
    // should have direct child's matches
    expect(containsForCluster).not.toContain('SERVICE_ACCOUNT');
    // should have grandchild
    expect(containsForCluster).toContain('DEPLOYMENT');
    // should have grandchild's matches
    expect(containsForCluster).not.toContain('CONTROL');
});

it('gets Contains relationships for Image', () => {
    const containsForImage = getEntityTypesByRelationship('IMAGE', 'CONTAINS');
    // should have direct child
    expect(containsForImage).toContain('IMAGE_COMPONENT');
    // should not contain itself
    expect(containsForImage).not.toContain('IMAGE');
    // should have grandchild
    expect(containsForImage).toContain('IMAGE_CVE');
    // should not have granchild's extended matches
    expect(containsForImage).not.toContain('DEPLOYMENT');
});

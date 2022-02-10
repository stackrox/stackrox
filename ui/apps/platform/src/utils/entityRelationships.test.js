import intersection from 'lodash/intersection';

import entityTypes from 'constants/entityTypes';
import relationshipTypes from 'constants/relationshipTypes';
import useCaseTypes from 'constants/useCaseTypes';
import entityRelationships, {
    entityGroupMap,
    getEntityTypesByRelationship,
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

it('gets Matches relationships', () => {
    let matches = getEntityTypesByRelationship(
        entityTypes.CLUSTER,
        relationshipTypes.MATCHES,
        useCaseTypes.CONFIG_MANAGEMENT
    );
    expect(matches).toContain(entityTypes.CONTROL);

    matches = getEntityTypesByRelationship(
        entityTypes.CLUSTER,
        relationshipTypes.MATCHES,
        useCaseTypes.VULN_MANAGEMENT
    );
    expect(matches).not.toContain(entityTypes.CONTROL);

    // should also get extended matches
    matches = getEntityTypesByRelationship(
        entityTypes.CVE,
        relationshipTypes.MATCHES,
        useCaseTypes.VULN_MANAGEMENT
    );
    expect(matches).toContain(entityTypes.IMAGE);
    expect(matches).toContain(entityTypes.COMPONENT);
    expect(matches).toContain(entityTypes.DEPLOYMENT);
    expect(matches).not.toContain(entityTypes.NAMESPACE);
});

it('gets direct Children relationships', () => {
    let children = getEntityTypesByRelationship(
        entityTypes.CLUSTER,
        relationshipTypes.CHILDREN,
        useCaseTypes.CONFIG_MANAGEMENT
    );
    expect(children).toContain(entityTypes.NODE);
    expect(children).toContain(entityTypes.NAMESPACE);
    expect(children).toContain(entityTypes.ROLE);

    children = getEntityTypesByRelationship(
        entityTypes.CLUSTER,
        relationshipTypes.CHILDREN,
        useCaseTypes.VULN_MANAGEMENT
    );
    expect(children).toContain(entityTypes.NODE);
    expect(children).toContain(entityTypes.NAMESPACE);
    expect(children).not.toContain(entityTypes.ROLE);
});

it('gets direct Parents relationships', () => {
    const parents = getEntityTypesByRelationship(
        entityTypes.CLUSTER,
        relationshipTypes.PARENTS,
        useCaseTypes.CONFIG_MANAGEMENT
    );
    expect(parents).toEqual([]);
});

it('gets Contain relationships', () => {
    let contains = getEntityTypesByRelationship(
        entityTypes.CLUSTER,
        relationshipTypes.CONTAINS,
        useCaseTypes.CONFIG_MANAGEMENT
    );

    // should have direct child
    expect(contains).toContain(entityTypes.ROLE);
    // should have direct child's matches
    expect(contains).toContain(entityTypes.SERVICE_ACCOUNT);
    // should have grandchild
    expect(contains).toContain(entityTypes.DEPLOYMENT);
    // should have grandchild's matches
    expect(contains).toContain(entityTypes.POLICY);
    expect(contains).toContain(entityTypes.CONTROL);

    contains = getEntityTypesByRelationship(
        entityTypes.CLUSTER,
        relationshipTypes.CONTAINS,
        useCaseTypes.VULN_MANAGEMENT
    );

    // should have direct child
    expect(contains).not.toContain(entityTypes.ROLE);
    expect(contains).toContain(entityTypes.NAMESPACE);
    // should have direct child's matches
    expect(contains).not.toContain(entityTypes.SERVICE_ACCOUNT);
    // should have grandchild
    expect(contains).toContain(entityTypes.DEPLOYMENT);
    // should have grandchild's matches
    expect(contains).toContain(entityTypes.POLICY);
    expect(contains).not.toContain(entityTypes.CONTROL);

    contains = getEntityTypesByRelationship(
        entityTypes.IMAGE,
        relationshipTypes.CONTAINS,
        useCaseTypes.VULN_MANAGEMENT
    );
    // should have direct child
    expect(contains).toContain(entityTypes.COMPONENT);
    // should not contain itself
    expect(contains).not.toContain(entityTypes.IMAGE);
    // should have grandchild
    expect(contains).toContain(entityTypes.CVE);
    // should not have granchild's extended matches
    expect(contains).not.toContain(entityTypes.DEPLOYMENT);
});

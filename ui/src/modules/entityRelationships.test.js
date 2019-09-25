import entityTypes from 'constants/entityTypes';
import relationshipTypes from 'constants/relationshipTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { getEntityTypesByRelationship } from './entityRelationships';

it('gets direct Matches relationships', () => {
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
    expect(children).not.toContain(entityTypes.NODE);
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
});

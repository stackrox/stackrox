export type RelationshipType = 'CONTAINS' | 'MATCHES' | 'PARENTS' | 'CHILDREN';

const relationshipTypes: Record<RelationshipType, RelationshipType> = {
    CONTAINS: 'CONTAINS',
    MATCHES: 'MATCHES',
    PARENTS: 'PARENTS',
    CHILDREN: 'CHILDREN',
};

export default relationshipTypes;

import entityRelationships from 'utils/entityRelationships';

function removeEntityContextColumns(columns, workflowState) {
    const entityContext = workflowState.getEntityContext();

    // For example, when entity context includes namespace:
    // Remove namespace column
    // Remove cluster column because it is parent of namespace

    return columns.filter(
        ({ entityType }) =>
            !entityContext[entityType] &&
            !Object.keys(entityContext).some((entityTypeInContext) =>
                entityRelationships.isParent(entityType, entityTypeInContext)
            )
    );
}

export default removeEntityContextColumns;

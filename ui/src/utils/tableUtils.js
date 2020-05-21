function removeEntityContextColumns(columns, workflowState) {
    const entityContext = workflowState.getEntityContext();
    return columns.filter((col) => !entityContext[col.entityType]);
}

export default removeEntityContextColumns;

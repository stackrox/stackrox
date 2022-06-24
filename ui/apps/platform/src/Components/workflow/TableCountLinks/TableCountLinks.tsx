import React, { ReactElement, useContext } from 'react';

import entityTypes, { ResourceType, resourceTypes } from 'constants/entityTypes';
import TableCountLink from 'Components/workflow/TableCountLink';
import workflowStateContext from 'Containers/workflowStateContext';

type TableCountLinksProps = {
    row: {
        vulnerabilityTypes: string[];
        deploymentCount: number;
        imageCount: number;
        componentCount: number;
        nodeCount?: number;
        id: string;
    };
    textOnly: boolean;
};

function TableCountLinks({ row, textOnly }: TableCountLinksProps): ReactElement {
    const workflowState = useContext(workflowStateContext);
    const entityType = workflowState.getCurrentEntityType();
    const entityContext = workflowState.getEntityContext() as Record<ResourceType, string>;
    const {
        vulnerabilityTypes,
        deploymentCount,
        imageCount,
        componentCount,
        nodeCount = 0,
        id,
    } = row;

    // TODO: refactor check for vulnerability types in follow-up PR
    const isImageVuln = vulnerabilityTypes?.includes('IMAGE_CVE');
    const isNodeVuln = entityType === entityTypes.NODE_CVE;

    // Only show entity counts on relevant pages. Node count is not currently supported.
    return (
        <div className="flex-col">
            {isImageVuln &&
                !entityContext[resourceTypes.DEPLOYMENT] &&
                !entityContext[resourceTypes.NODE] && (
                    <TableCountLink
                        entityType={resourceTypes.DEPLOYMENT}
                        count={deploymentCount}
                        textOnly={textOnly}
                        selectedRowId={id}
                    />
                )}
            {isImageVuln &&
                !entityContext[resourceTypes.IMAGE] &&
                !entityContext[resourceTypes.NODE] && (
                    <TableCountLink
                        entityType={resourceTypes.IMAGE}
                        count={imageCount}
                        textOnly={textOnly}
                        selectedRowId={id}
                    />
                )}
            {isNodeVuln && !entityContext[resourceTypes.NODE] && (
                <TableCountLink
                    entityType={resourceTypes.NODE}
                    count={nodeCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
            {/* TODO: strengthen check for COMPONENT context to distinguish check
                between IMAGE_COMPONENT and NODE_COMPONENT in later PR */}
            {!isNodeVuln && !entityContext[resourceTypes.COMPONENT] && (
                <TableCountLink
                    entityType={resourceTypes.COMPONENT}
                    count={componentCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
        </div>
    );
}

export default TableCountLinks;

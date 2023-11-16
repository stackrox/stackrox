import React, { ReactElement, useContext } from 'react';

import entityTypes, { ResourceType, resourceTypes } from 'constants/entityTypes';
import TableCountLink from 'Components/workflow/TableCountLink';
import workflowStateContext from 'Containers/workflowStateContext';
import fixableVulnTypeContext from 'Containers/VulnMgmt/fixableVulnTypeContext';

type TableCountLinksProps = {
    row: {
        vulnerabilityTypes: string[];
        deploymentCount: number;
        imageCount: number;
        componentCount: number;
        nodeCount?: number;
        clusterCount?: number;
        id: string;
    };
    textOnly: boolean;
};

function TableCountLinks({ row, textOnly }: TableCountLinksProps): ReactElement {
    const fixableVulnType: string | null | undefined = useContext(fixableVulnTypeContext);
    const workflowState = useContext(workflowStateContext);
    const entityType = workflowState.getCurrentEntityType();
    const entityContext = workflowState.getEntityContext() as Record<ResourceType, string>;
    const {
        deploymentCount,
        imageCount,
        componentCount,
        nodeCount = 0,
        clusterCount = 0,
        id,
    } = row;

    // TODO: refactor check for vulnerability types in follow-up PR
    const isLegacyVuln = entityType === entityTypes.CVE;
    const isImageVuln =
        entityType === entityTypes.IMAGE_CVE || fixableVulnType === entityTypes.IMAGE_CVE;
    const isNodeVuln =
        entityType === entityTypes.NODE_CVE || fixableVulnType === entityTypes.NODE_CVE;
    const isClusterVuln =
        entityType === entityTypes.CLUSTER_CVE || fixableVulnType === entityTypes.CLUSTER_CVE;

    // Only show entity counts on relevant pages. Node count is not currently supported.
    return (
        <div className="flex-col">
            {(isImageVuln || isLegacyVuln) &&
                !entityContext[resourceTypes.DEPLOYMENT] &&
                !entityContext[resourceTypes.NODE] && (
                    <TableCountLink
                        entityType={resourceTypes.DEPLOYMENT}
                        count={deploymentCount}
                        textOnly={textOnly}
                        selectedRowId={id}
                    />
                )}
            {(isImageVuln || isLegacyVuln) &&
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
            {!isImageVuln &&
                !isClusterVuln &&
                !isNodeVuln &&
                !entityContext[resourceTypes.COMPONENT] && (
                    <TableCountLink
                        entityType={resourceTypes.COMPONENT}
                        count={componentCount}
                        textOnly={textOnly}
                        selectedRowId={id}
                    />
                )}
            {isImageVuln && !entityContext[resourceTypes.IMAGE_COMPONENT] && (
                <TableCountLink
                    entityType={resourceTypes.IMAGE_COMPONENT}
                    count={componentCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
            {isNodeVuln && !entityContext[resourceTypes.NODE_COMPONENT] && (
                <TableCountLink
                    entityType={resourceTypes.NODE_COMPONENT}
                    count={componentCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
            {isClusterVuln && !entityContext[resourceTypes.CLUSTER] && (
                <TableCountLink
                    entityType={resourceTypes.CLUSTER}
                    count={clusterCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
        </div>
    );
}

export default TableCountLinks;

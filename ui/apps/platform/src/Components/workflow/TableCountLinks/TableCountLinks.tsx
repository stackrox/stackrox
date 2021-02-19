import React, { ReactElement, useContext } from 'react';

import { ResourceType, resourceTypes } from 'constants/entityTypes';
import TableCountLink from 'Components/workflow/TableCountLink';
import workflowStateContext from 'Containers/workflowStateContext';

type TableCountLinksProps = {
    row: {
        vulnerabilityTypes: string[];
        deploymentCount: number;
        imageCount: number;
        componentCount: number;
        id: string;
    };
    textOnly: boolean;
};

function TableCountLinks({ row, textOnly }: TableCountLinksProps): ReactElement {
    const workflowState = useContext(workflowStateContext);
    const entityContext = workflowState.getEntityContext() as Record<ResourceType, string>;
    const { vulnerabilityTypes, deploymentCount, imageCount, componentCount, id } = row;

    const isImageVuln = vulnerabilityTypes.includes('IMAGE_CVE');

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
            {!entityContext[resourceTypes.COMPONENT] && (
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

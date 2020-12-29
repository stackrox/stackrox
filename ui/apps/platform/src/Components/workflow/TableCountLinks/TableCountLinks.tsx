import React, { ReactElement, useContext } from 'react';

import { ResourceType, resourceTypes } from 'constants/entityTypes';
import TableCountLink from 'Components/workflow/TableCountLink';
import workflowStateContext from 'Containers/workflowStateContext';

type TableCountLinksProps = {
    row: {
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
    const { deploymentCount, imageCount, componentCount, id } = row;

    return (
        <div className="flex-col">
            {!entityContext[resourceTypes.DEPLOYMENT] && (
                <TableCountLink
                    entityType={resourceTypes.DEPLOYMENT}
                    count={deploymentCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
            {!entityContext[resourceTypes.IMAGE] && (
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

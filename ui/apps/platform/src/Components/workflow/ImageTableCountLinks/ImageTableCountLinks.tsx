import React, { ReactElement, useContext } from 'react';

import { ResourceType, resourceTypes } from 'constants/entityTypes';
import TableCountLink from 'Components/workflow/TableCountLink';
import workflowStateContext from 'Containers/workflowStateContext';

type ImageTableCountLinksProps = {
    row: {
        deploymentCount: number;
        componentCount: number;
        id: string;
    };
    textOnly: boolean;
};

function ImageTableCountLinks({ row, textOnly }: ImageTableCountLinksProps): ReactElement {
    const workflowState = useContext(workflowStateContext);
    const entityContext = workflowState.getEntityContext() as Record<ResourceType, string>;

    const { deploymentCount, componentCount, id } = row;

    // Only show entity counts on relevant pages.
    return (
        <div className="flex-col">
            <TableCountLink
                entityType={resourceTypes.DEPLOYMENT}
                count={deploymentCount}
                textOnly={textOnly}
                selectedRowId={id}
            />
            {!entityContext[resourceTypes.IMAGE_COMPONENT] && (
                <TableCountLink
                    entityType={resourceTypes.IMAGE_COMPONENT}
                    count={componentCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
        </div>
    );
}

export default ImageTableCountLinks;

import React, { ReactElement, useContext } from 'react';

import entityTypes from 'constants/entityTypes';
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
    const entityContext = workflowState.getEntityContext();
    const { deploymentCount, imageCount, componentCount, id } = row;

    return (
        <div className="flex-col">
            {!entityContext[entityTypes.DEPLOYMENT] && (
                <TableCountLink
                    entityType={entityTypes.DEPLOYMENT}
                    count={deploymentCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
            {!entityContext[entityTypes.IMAGE] && (
                <TableCountLink
                    entityType={entityTypes.IMAGE}
                    count={imageCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
            {!entityContext[entityTypes.COMPONENT] && (
                <TableCountLink
                    entityType={entityTypes.COMPONENT}
                    count={componentCount}
                    textOnly={textOnly}
                    selectedRowId={id}
                />
            )}
        </div>
    );
}

export default TableCountLinks;

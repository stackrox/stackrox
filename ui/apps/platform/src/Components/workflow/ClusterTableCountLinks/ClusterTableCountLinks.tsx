import React, { ReactElement } from 'react';

import { resourceTypes } from 'constants/entityTypes';
import TableCountLink from 'Components/workflow/TableCountLink';

type ClusterTableCountLinksProps = {
    row: {
        namespaceCount: number;
        deploymentCount: number;
        nodeCount: number;
        id: string;
    };
    textOnly: boolean;
};

function ClusterTableCountLinks({ row, textOnly }: ClusterTableCountLinksProps): ReactElement {
    const { deploymentCount, namespaceCount, nodeCount, id } = row;

    // Only show entity counts on relevant pages. Node count is not currently supported.
    return (
        <div className="flex-col">
            <TableCountLink
                entityType={resourceTypes.NAMESPACE}
                count={namespaceCount}
                textOnly={textOnly}
                selectedRowId={id}
            />
            <TableCountLink
                entityType={resourceTypes.DEPLOYMENT}
                count={deploymentCount}
                textOnly={textOnly}
                selectedRowId={id}
            />
            <TableCountLink
                entityType={resourceTypes.NODE}
                count={nodeCount}
                textOnly={textOnly}
                selectedRowId={id}
            />
        </div>
    );
}

export default ClusterTableCountLinks;

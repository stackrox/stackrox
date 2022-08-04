import React, { ReactElement, useState } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import pluralize from 'pluralize';

import { selectors } from 'reducers';
import useNavigateToEntity from 'Containers/Network/SidePanel/useNavigateToEntity';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NamespaceDeploymentsTable from './NamespaceDeploymentsTable';

type NamespaceDeploymentsProps = {
    deployments: [];
    filterState?: number;
};

function NamespaceDeployments({
    deployments,
    filterState = 0,
}: NamespaceDeploymentsProps): ReactElement {
    const onNavigateToEntity = useNavigateToEntity();

    const [page, setPage] = useState(0);
    const subHeaderText = `${deployments.length} ${pluralize('Deployment', deployments.length)}`;

    return (
        <PanelNew testid="panel">
            <PanelHead>
                <PanelTitle testid="panel-header" text={subHeaderText} />
                <PanelHeadEnd>
                    <TablePagination
                        page={page}
                        dataLength={deployments?.length}
                        setPage={setPage}
                    />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <NamespaceDeploymentsTable
                    deployments={deployments}
                    page={page}
                    onNavigateToDeploymentById={onNavigateToEntity}
                    filterState={filterState}
                />
            </PanelBody>
        </PanelNew>
    );
}

const mapStateToProps = createStructuredSelector({
    filterState: selectors.getNetworkGraphFilterMode,
    networkGraphRef: selectors.getNetworkGraphRef,
});

export default connect(mapStateToProps, null)(NamespaceDeployments);

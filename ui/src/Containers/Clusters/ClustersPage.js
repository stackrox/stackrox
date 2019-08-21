import React from 'react';

import PageHeader from 'Components/PageHeader';
import Panel from 'Components/Panel';

const ClustersPage = () => {
    // @TODO: flesh out the new Clusters page layout, placeholders for now
    const paginationComponent = <div>Pagination component</div>;

    const headerComponent = <div>Panel text header component</div>;

    return (
        <section className="flex flex-1 flex-col h-full">
            <PageHeader header="Clusters" />
            <div className="flex flex-1 flex-col">
                <div className="flex flex-1 relative">
                    <div className="shadow border-primary-300 bg-base-100 w-full overflow-hidden">
                        <Panel
                            headerTextComponent={headerComponent}
                            headerComponents={paginationComponent}
                        >
                            <div className="w-full">Clusters table goes here</div>
                        </Panel>
                    </div>
                </div>
            </div>
        </section>
    );
};

export default ClustersPage;

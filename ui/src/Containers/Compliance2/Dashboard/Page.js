import React from 'react';
import { Link } from 'react-router-dom';

import StandardsAcrossEntity from 'Containers/Compliance2/widgets/StandardsAcrossEntity';
import StandardsByEntity from 'Containers/Compliance2/widgets/StandardsByEntity';
import LinkListWidget from 'Components/widgets/LinkListWidget';
import Button from 'Components/Button';

import { horizontalBarData } from 'mockData/graphDataMock';
import entityTypes from 'constants/entityTypes';

import DashboardHeader from './Header';

const namespacesList = [
    { name: 'namespace-1', link: '/main/compliance2/namespace/1' },
    { name: 'namespace-2', link: '/main/compliance2/namespace/2' },
    { name: 'namespace-3', link: '/main/compliance2/namespace/3' },
    { name: 'namespace-4', link: '/main/compliance2/namespace/4' },
    { name: 'namespace-5', link: '/main/compliance2/namespace/5' }
];

// Just fixing a merge conflict. Not really sure what button onClick should do
function doNothing() {}

const ComplianceDashboardPage = () => (
    <section className="flex flex-1 flex-col h-full">
        <div className="flex flex-1 flex-col">
            <DashboardHeader />
            <div className="flex-1 relative bg-base-200 p-4 overflow-auto">
                <div className="grid xl:grid-columns-3 md:grid-columns-2 sm:grid-columns-1 grid-gap-6">
                    <StandardsAcrossEntity type={entityTypes.CLUSTERS} data={horizontalBarData} />
                    <LinkListWidget
                        title="5 Related Namespaces"
                        data={namespacesList}
                        headerComponents={
                            <Link className="no-underline" to="/main/compliance2/namespaces">
                                <Button
                                    className="btn-sm btn-base"
                                    onClick={doNothing}
                                    text="View All"
                                />
                            </Link>
                        }
                    />
                    <StandardsByEntity type={entityTypes.CLUSTERS} />
                </div>
            </div>
        </div>
    </section>
);

export default ComplianceDashboardPage;

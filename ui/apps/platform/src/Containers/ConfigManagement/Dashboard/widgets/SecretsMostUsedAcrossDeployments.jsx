import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import dateFns from 'date-fns';

import Loader from 'Components/Loader';
import URLService from 'utils/URLService';
import entityTypes from 'constants/entityTypes';
import Query from 'Components/ThrowingQuery';
import Widget from 'Components/Widget';
import useWorkflowMatch from 'hooks/useWorkflowMatch';

const QUERY = gql`
    query secrets {
        secrets {
            id
            name
            clusterName
            namespace
            files {
                name
                type
                metadata {
                    __typename
                    ... on Cert {
                        endDate
                        startDate
                    }
                    ... on ImagePullSecret {
                        registries {
                            name
                            username
                        }
                    }
                }
            }
            deploymentCount
        }
    }
`;

const getCertificateStatus = (files) => {
    let status = 'no';
    files.forEach((file) => {
        if (file.metadata) {
            const { startDate, endDate } = file.metadata;
            if (!startDate && !endDate) {
                return;
            }

            const today = new Date().toISOString();
            const isUpcoming = dateFns.isAfter(startDate, today);
            const hasExpired = dateFns.isAfter(today, endDate);

            if (isUpcoming) {
                status = 'upcoming';
            } else if (hasExpired) {
                status = 'expired';
            } else {
                status = 'valid';
            }
        }
    });

    return `has ${status} certs`;
};

const SecretsMostUsedAcrossDeployments = () => {
    const match = useWorkflowMatch();
    const location = useLocation();

    function processData(data) {
        if (!data || !data.secrets) {
            return [];
        }

        return data.secrets
            .filter((secret) => secret.deploymentCount)
            .sort((a, b) => b.deploymentCount - a.deploymentCount)
            .slice(0, 10);
    }

    return (
        <Query query={QUERY}>
            {({ loading, data }) => {
                let contents = <Loader />;
                const viewAllURL = URLService.getURL(match, location)
                    .base(entityTypes.SECRET)
                    .url();

                const viewAllLink = (
                    <Link to={viewAllURL} className="no-underline btn-sm btn-base">
                        View all
                    </Link>
                );

                if (!loading && data) {
                    const results = processData(data);

                    contents = (
                        <ul
                            className="w-full columns-2 columns-gap-0"
                            style={{ columnRule: '1px solid var(--base-300)' }}
                        >
                            {results.map((item, index) => {
                                const linkTo = URLService.getURL(match, location)
                                    .base(entityTypes.SECRET)
                                    .push(item.id)
                                    .url();
                                return (
                                    <li
                                        key={item.id}
                                        className={`inline-block flex flex-row border-base-300 w-full ${
                                            index !== 4 || index !== 9 ? 'border-b' : ''
                                        }`}
                                    >
                                        <div className="self-center text-2xl pl-4 pr-4">
                                            {index + 1}
                                        </div>
                                        <div className="flex flex-col truncate pr-4 pb-4 pt-4">
                                            <span className="text-sm">
                                                {item.clusterName}/{item.namespace}
                                            </span>
                                            <Link to={linkTo}>{item.name}</Link>
                                            {item.deploymentCount > 0 && (
                                                <span className="truncate text-sm">
                                                    {`${item.deploymentCount} ${pluralize(
                                                        'deployment',
                                                        item.deploymentCount
                                                    )}, `}
                                                    {getCertificateStatus(item.files)}
                                                </span>
                                            )}
                                        </div>
                                    </li>
                                );
                            })}
                        </ul>
                    );
                }
                return (
                    <Widget
                        className="s-2 overflow-hidden pdf-page"
                        header="Secrets most used across deployments"
                        headerComponents={viewAllLink}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

export default SecretsMostUsedAcrossDeployments;

import React from 'react';
import gql from 'graphql-tag';
import Loader from 'Components/Loader';
import { Link, withRouter } from 'react-router-dom';
import URLService from 'utils/URLService';
import entityTypes from 'constants/entityTypes';
import networkStatuses from 'constants/networkStatuses';
import Query from 'Components/ThrowingQuery';
import Widget from 'Components/Widget';
import Lollipop from 'Components/visuals/Lollipop';
import ReactRouterPropTypes from 'react-router-prop-types';

const QUERY = gql`
    query usersWithClusterAdminRoles {
        clusters {
            id
            subjects {
                subject {
                    name
                }
                type
                clusterAdmin
            }
        }
    }
`;

const UsersWithMostClusterAdminRoles = ({ match, location }) => {
    function processData(data) {
        if (!data || !data.clusters) return [];

        const subjectCounts = data.clusters.reduce((allSubjects, cluster) => {
            if (!cluster.subjects) return allSubjects;
            const newSubjects = { ...allSubjects };

            cluster.subjects
                .filter((subject) => subject.clusterAdmin)
                .forEach((subject) => {
                    const { name } = subject.subject;
                    if (!allSubjects[name]) newSubjects[name] = 0;

                    newSubjects[name] += 1;
                });
            return newSubjects;
        }, {});

        return Object.entries(subjectCounts)
            .map((entry) => {
                const link = URLService.getURL(match, location)
                    .base(entityTypes.SUBJECT)
                    .push(entry[0])
                    .url();
                return {
                    y: entry[0],
                    x: entry[1],
                    hint: {
                        title: entry[0],
                        body: entry[1],
                    },
                    link,
                };
            })
            .sort((a, b) => b.x - a.x)
            .slice(0, 6);
    }

    return (
        <Query query={QUERY}>
            {({ loading, data, networkStatus }) => {
                let contents = <Loader />;
                let viewAllLink;

                if (!loading && data && networkStatus === networkStatuses.READY) {
                    const results = processData(data);

                    const linkTo = URLService.getURL(match, location)
                        .base(entityTypes.SUBJECT)
                        .url();

                    viewAllLink = (
                        <Link to={linkTo} className="no-underline">
                            <button className="btn-sm btn-base" type="button">
                                View All
                            </button>
                        </Link>
                    );

                    contents = <Lollipop data={results} />;
                }
                return (
                    <Widget
                        className="s-2 overflow-hidden pdf-page"
                        header="Users with most Cluster Admin Roles"
                        headerComponents={viewAllLink}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

UsersWithMostClusterAdminRoles.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
};

export default withRouter(UsersWithMostClusterAdminRoles);

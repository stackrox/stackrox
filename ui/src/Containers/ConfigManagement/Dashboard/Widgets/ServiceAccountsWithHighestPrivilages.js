import React from 'react';
import gql from 'graphql-tag';
import Loader from 'Components/Loader';
import { Link, withRouter } from 'react-router-dom';
import { Tooltip } from 'react-tippy';
import URLService from 'modules/URLService';
import entityTypes from 'constants/entityTypes';
import networkStatuses from 'constants/networkStatuses';
import Query from 'Components/ThrowingQuery';
import Widget from 'Components/Widget';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import PermissionCounts from 'Containers/ConfigManagement/Entity/widgets/PermissionCounts';

const QUERY = gql`
    query serviceAccounts {
        serviceAccounts {
            id
            name
            scopedPermissions {
                scope
                permissions {
                    key
                    values
                }
            }
        }
    }
`;

const PermissionsText = ({ serviceAccount }) => {
    if (serviceAccount.scopedPermissions && serviceAccount.scopedPermissions.length === 0)
        return 'No permissions';
    return (
        <div className="truncate italic">
            <PermissionCounts scopedPermissions={serviceAccount.scopedPermissions} />
        </div>
    );
};

PermissionsText.propTypes = {
    serviceAccount: PropTypes.shape({}).isRequired
};

const ServiceAccountsWithHighestPrivilages = ({ match, location }) => {
    function processData(data) {
        return data;
    }

    return (
        <Query query={QUERY}>
            {({ loading, data, networkStatus }) => {
                let contents = <Loader />;
                let viewAllLink;

                if (!loading && data && networkStatus === networkStatuses.READY) {
                    const results = processData(data);

                    const slicedData = results.serviceAccounts.slice(0, 10);
                    const linkTo = URLService.getURL(match, location)
                        .base(entityTypes.SERVICE_ACCOUNT)
                        .url();

                    viewAllLink = (
                        <Link to={linkTo} className="no-underline">
                            <button className="btn-sm btn-base" type="button">
                                View All
                            </button>
                        </Link>
                    );

                    contents = (
                        <ul className="list-reset w-full columns-2 columns-gap-0">
                            {slicedData.map((item, index) => (
                                <Link
                                    key={`${item.id}-${index}`}
                                    to={`${linkTo}/${item.id}`}
                                    className="no-underline text-base-600"
                                >
                                    <li key={`${item.name}-${index}`} className="hover:bg-base-200">
                                        <div
                                            className={`flex flex-row border-base-400 ${
                                                index !== 4 && index !== 9 ? 'border-b' : ''
                                            } ${index < 5 ? 'border-r' : ''}`}
                                        >
                                            <div className="self-center text-2xl tracking-widest pl-4 pr-4">
                                                {index + 1}
                                            </div>
                                            <div className="flex flex-col truncate pr-4 pb-4 pt-4 text-sm">
                                                <span className="pb-2">{item.name}</span>
                                                <Tooltip
                                                    position="top"
                                                    trigger="mouseenter"
                                                    animation="none"
                                                    duration={0}
                                                    arrow
                                                    distance={20}
                                                    html={
                                                        <span className="text-sm italic">
                                                            <PermissionsText
                                                                serviceAccount={item}
                                                            />
                                                        </span>
                                                    }
                                                    unmountHTMLWhenHide
                                                >
                                                    {item.scopedPermissions &&
                                                        item.scopedPermissions.length === 0 && (
                                                            <span className="italic">
                                                                No Permissions
                                                            </span>
                                                        )}
                                                    {item.scopedPermissions &&
                                                        item.scopedPermissions.length >= 1 && (
                                                            <PermissionsText
                                                                serviceAccount={item}
                                                            />
                                                        )}
                                                </Tooltip>
                                            </div>
                                        </div>
                                    </li>
                                </Link>
                            ))}
                        </ul>
                    );
                }
                return (
                    <Widget
                        className="s-2 overflow-hidden pdf-page"
                        header="Service Accounts with Highest privileges"
                        headerComponents={viewAllLink}
                    >
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ServiceAccountsWithHighestPrivilages.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(ServiceAccountsWithHighestPrivilages);

import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { licenseType } from 'mockData/licenseData';

import PageHeader from 'Components/PageHeader';

import LicenseExpiration from './widgets/LicenseExpiration';
import UpgradeSupport from './widgets/UpgradeSupport';

const Page = ({ license }) => {
    const customerID = license ? license.license.metadata.licensedForId : '';
    const header = `License: ${licenseType}`;
    const subHeader = `Customer ID: #${customerID}`;
    return (
        <section className="flex flex-1 h-full w-full">
            <div className="flex flex-1 flex-col w-full">
                <PageHeader header={header} subHeader={subHeader} />
                <div
                    className="flex-1 relative p-6 xxxl:p-8 bg-base-200"
                    style={{ '--start': '#d3d9ff', '--end': '#b9dbff' }}
                >
                    <div
                        className="grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense"
                        style={{ '--min-tile-height': '160px' }}
                    >
                        <LicenseExpiration />
                        <UpgradeSupport />
                    </div>
                </div>
            </div>
        </section>
    );
};

Page.propTypes = {
    license: PropTypes.shape({})
};

Page.defaultProps = {
    license: null
};

const mapStateToProps = createStructuredSelector({
    license: selectors.getLicense
});

export default connect(
    mapStateToProps,
    null
)(Page);

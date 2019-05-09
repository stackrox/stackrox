import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import PageHeader from 'Components/PageHeader';
import Form from './Form';

const mockInitialValues = {
    header: false,
    headerText: 'hello this is sample header text',
    headerTextColor: 'red',
    headerBackgroundColor: 'pink',
    footer: false,
    footerTextColor: 'blue',
    footerBackgroundColor: 'black'
};

const Page = () => (
    <section className="flex flex-1 h-full w-full">
        <div className="flex flex-1 flex-col w-full">
            <PageHeader header="System Configuration" />
            <div className="w-full h-full flex pb-0 bg-base-200">
                <Form initialValues={mockInitialValues} />
            </div>
        </div>
    </section>
);

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

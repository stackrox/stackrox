import React from 'react';
import PropTypes from 'prop-types';
import { DndProvider } from 'react-dnd';
import Backend from 'react-dnd-html5-backend';
import { FieldArray, reduxForm } from 'redux-form';
import { connect } from 'react-redux';

import PolicyBuilderKeys from 'Components/PolicyBuilderKeys';
import PolicySections from './PolicySections';
import { policyConfiguration } from './descriptors';

function BooleanPolicySection({ readOnly, hasHeader }) {
    if (readOnly)
        return (
            <div className="w-full flex">
                <FieldArray
                    name="policySections"
                    component={PolicySections}
                    hasHeader={hasHeader}
                    readOnly
                    className="w-full"
                />
            </div>
        );
    return (
        <DndProvider backend={Backend}>
            <div className="w-full flex">
                <FieldArray name="policySections" component={PolicySections} />
                <PolicyBuilderKeys keys={policyConfiguration.descriptor} />
            </div>
        </DndProvider>
    );
}

BooleanPolicySection.propTypes = {
    readOnly: PropTypes.bool,
    hasHeader: PropTypes.bool,
};

BooleanPolicySection.defaultProps = {
    readOnly: false,
    hasHeader: true,
};

export default reduxForm({
    form: 'policyCreationForm',
    enableReinitialize: true,
    destroyOnUnmount: false,
})(connect(null)(BooleanPolicySection));

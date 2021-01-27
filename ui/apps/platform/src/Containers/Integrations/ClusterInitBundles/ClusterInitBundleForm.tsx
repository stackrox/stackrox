import React from 'react';
import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';

import { clusterInitBundleFormId } from 'reducers/clusterInitBundles';
import FormField from 'Components/FormField';
import ReduxTextField from 'Components/forms/ReduxTextField';

const ClusterInitBundleForm = () => {
    return (
        <form className="p-4 w-full mb-8" data-testid="bootstrap-token-form">
            <FormField label="Cluster Init Bundle Name" required>
                <ReduxTextField name="name" />
            </FormField>
        </form>
    );
};

const ConnectedForm = connect()(
    reduxForm({ form: clusterInitBundleFormId })(ClusterInitBundleForm)
);

export default ConnectedForm;

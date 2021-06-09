import React from 'react';
import { Grid, GridItem } from '@patternfly/react-core';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { reduxForm } from 'redux-form';

import ConfigBannerFormWidget from './ConfigBannerFormWidget';
import ConfigDataRetentionFormWidget from './ConfigDataRetentionFormWidget';
import ConfigLoginFormWidget from './ConfigLoginFormWidget';
import { pageLayoutClassName } from './SystemConfig.constants';
import ConfigTelemetryDetailWidget from './ConfigTelemetryDetailWidget';

const Form = ({ initialValues, onSubmit, config }) => (
    <form className={pageLayoutClassName} initialvalues={initialValues} onSubmit={onSubmit}>
        <Grid hasGutter>
            <GridItem span={12}>
                <ConfigDataRetentionFormWidget
                    initialValues={initialValues}
                    privateConfig={config.privateConfig}
                />
            </GridItem>
            <GridItem span={6}>
                <ConfigBannerFormWidget type="header" />
            </GridItem>
            <GridItem span={6}>
                <ConfigBannerFormWidget type="footer" />
            </GridItem>
            <GridItem span={6}>
                <ConfigLoginFormWidget />
            </GridItem>
            <GridItem span={6}>
                <ConfigTelemetryDetailWidget config={config.telemetryConfig} editable />
            </GridItem>
        </Grid>
    </form>
);

Form.propTypes = {
    config: PropTypes.shape({
        privateConfig: PropTypes.shape({
            alertConfig: PropTypes.shape({}),
            imageRetentionDurationDays: PropTypes.number,
        }),
        telemetryConfig: PropTypes.shape({
            enabled: PropTypes.bool,
        }),
    }).isRequired,
    onSubmit: PropTypes.func.isRequired,
    initialValues: PropTypes.shape({}),
};

Form.defaultProps = {
    initialValues: null,
};

export default reduxForm({
    form: 'system-config-form',
})(connect(null, null)(Form));

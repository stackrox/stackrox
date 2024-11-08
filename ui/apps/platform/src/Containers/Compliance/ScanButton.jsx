import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Mutation } from '@apollo/client/react/components';
import * as Icon from 'react-feather';
import { Spinner } from '@patternfly/react-core';

import { actions as notificationActions } from 'reducers/notifications';
import { TRIGGER_SCAN } from 'queries/standard';
import Button from 'Components/Button';

class ScanButton extends React.Component {
    static propTypes = {
        className: PropTypes.string,
        text: PropTypes.string.isRequired,
        textCondensed: PropTypes.string,
        textClass: PropTypes.string,
        clusterId: PropTypes.string,
        standardId: PropTypes.string,
        loaderSize: PropTypes.number,
        addToast: PropTypes.func.isRequired,
        removeToast: PropTypes.func.isRequired,
        onScanTriggered: PropTypes.func,
        scanInProgress: PropTypes.bool,
    };

    static defaultProps = {
        className: 'btn btn-base h-10',
        clusterId: '*',
        textClass: null,
        textCondensed: null,
        standardId: '*',
        loaderSize: 20,
        onScanTriggered: () => {},
    };

    onClick = (triggerScan) => () => {
        const { clusterId, standardId } = this.props;
        triggerScan({ variables: { clusterId, standardId } })
            .then(() => {
                this.props.onScanTriggered();
            })
            .catch((e) => {
                this.props.addToast(e.message);
                setTimeout(this.props.removeToast, 2000);
            });
    };

    render() {
        const { className, text, textCondensed, textClass, loaderSize, scanInProgress } =
            this.props;

        return (
            <Mutation mutation={TRIGGER_SCAN}>
                {(triggerScan, { loading }) => {
                    return (
                        <Button
                            dataTestId="scan-button"
                            className={className}
                            text={text}
                            textCondensed={textCondensed}
                            textClass={textClass}
                            icon={
                                scanInProgress ? (
                                    <Spinner size="md" className="mx-1 lg:ml-1 lg:mr-3" />
                                ) : (
                                    <Icon.RefreshCcw
                                        size="14"
                                        className="bg-base-100 mx-1 lg:ml-1 lg:mr-3"
                                    />
                                )
                            }
                            onClick={this.onClick(triggerScan)}
                            isLoading={loading}
                            disabled={loading}
                            loaderSize={loaderSize}
                        />
                    );
                }}
            </Mutation>
        );
    }
}

const mapDispatchToProps = {
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
};

export default connect(null, mapDispatchToProps)(ScanButton);

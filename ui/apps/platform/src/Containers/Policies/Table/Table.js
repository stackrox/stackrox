import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';

import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as tableActions } from 'reducers/policies/table';
import { actions as wizardActions } from 'reducers/policies/wizard';

import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import NoResultsMessage from 'Components/NoResultsMessage';
import TablePagination from 'Components/TablePagination';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import Buttons from 'Containers/Policies/Table/Buttons';
import TableContents from 'Containers/Policies/Table/TableContents';
import PolicyImportDialogue from 'Containers/Policies/Table/PolicyImportDialogue';

const POLICIES_TABLE_PAGE_SIZE = 150;

// Table is the heading line with options to reasses and add a policy, as well as the underlying policy
// rows.
class Table extends Component {
    static propTypes = {
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        page: PropTypes.number.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        setPage: PropTypes.func.isRequired,
        selectPolicyId: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,
        openWizard: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            showImportDialogue: false,
        };
    }

    setSelectedPolicy = (policyId) => {
        // Add policy to history.
        const urlSuffix = `/${policyId}`;
        this.props.history.push({
            pathname: `/main/policies${urlSuffix}`,
        });

        // Select the policy so that it is highlighted in the table.
        this.props.selectPolicyId(policyId);

        // Bring up the wizard with that policy.
        this.props.setWizardStage(wizardStages.details);
        this.props.openWizard();
    };

    startPolicyImport = () => {
        this.setState({ showImportDialogue: true });
    };

    closeImportDialogue = () => {
        this.setState({ showImportDialogue: false });
    };

    getTableHeaderText = () => {
        const selectionCount = this.props.selectedPolicyIds.length;
        const rowCount = this.props.policies.length;
        return selectionCount !== 0
            ? `${selectionCount} ${selectionCount === 1 ? 'Policy' : 'Policies'} Selected`
            : `${rowCount} ${rowCount === 1 ? 'Policy' : 'Policies'} ${
                  this.props.isViewFiltered ? 'Matched' : ''
              }`;
    };

    render() {
        if (!this.props.policies.length) {
            return <NoResultsMessage message="No results found. Please refine your search." />;
        }

        return (
            <div className="flex-shrink-1 overflow-hidden w-full">
                <PanelNew testid="panel">
                    <PanelHead>
                        <PanelTitle
                            isUpperCase
                            testid="panel-header"
                            text={this.getTableHeaderText()}
                        />
                        <PanelHeadEnd>
                            <Buttons startPolicyImport={this.startPolicyImport} />
                            <TablePagination
                                pageSize={POLICIES_TABLE_PAGE_SIZE}
                                page={this.props.page}
                                dataLength={this.props.policies.length}
                                setPage={this.props.setPage}
                            />
                            {this.state.showImportDialogue && (
                                <PolicyImportDialogue closeAction={this.closeImportDialogue} />
                            )}
                        </PanelHeadEnd>
                    </PanelHead>
                    <PanelBody>
                        <TableContents
                            pageSize={POLICIES_TABLE_PAGE_SIZE}
                            setSelectedPolicy={this.setSelectedPolicy}
                        />
                    </PanelBody>
                </PanelNew>
            </div>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getPoliciesSearchOptions],
    (searchOptions) => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    policies: selectors.getFilteredPolicies,
    page: selectors.getTablePage,
    isViewFiltered,
});

const mapDispatchToProps = {
    selectPolicyId: tableActions.selectPolicyId,
    setWizardStage: wizardActions.setWizardStage,
    openWizard: pageActions.openWizard,
    setPage: tableActions.setTablePage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(Table));

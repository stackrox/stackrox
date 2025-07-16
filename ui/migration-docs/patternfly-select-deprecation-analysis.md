# PatternFly Select Component Deprecation Analysis

## Overview

This document lists all files in the codebase that import Select-related components from the deprecated `@patternfly/react-core/deprecated` module. These imports need to be migrated to the modern `@patternfly/react-core` package.

## Migration Status

-   **Total files found: 56**
-   **Migration status: Not started**

## Deprecated Import Components

The following Select-related components are being imported from the deprecated module:

-   `Select`
-   `SelectOption`
-   `SelectProps`
-   `SelectGroup`
-   `SelectOptionObject`
-   `SelectOptionProps`

## Files Requiring Migration

### Violations (1 file)

-   [ ] `src/Containers/Violations/ViolationsTablePanel.tsx`

### Vulnerabilities (5 files)

-   [ ] `src/Containers/Vulnerabilities/VulnerablityReporting/forms/CollectionSelection.tsx`
-   [ ] `src/Containers/Vulnerabilities/VulnerablityReporting/forms/ReportParametersForm.tsx`
-   [ ] `src/Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/ReportJobs.tsx`
-   [ ] `src/Containers/Vulnerabilities/components/CVEStatusDropdown.tsx`
-   [ ] `src/Containers/Vulnerabilities/components/CVESeverityDropdown.tsx`

### Collections (4 files)

-   [ ] `src/Containers/Collections/CollectionResults.tsx`
-   [ ] `src/Containers/Collections/RuleSelector/ByNameSelector.tsx`
-   [ ] `src/Containers/Collections/RuleSelector/RuleSelector.tsx`
-   [ ] `src/Containers/Collections/RuleSelector/MatchTypeSelect.tsx`

### System Configuration (2 files)

-   [ ] `src/Containers/SystemConfig/Form/FormSelect.tsx`
-   [ ] `src/Containers/SystemConfig/Form/SystemConfigForm.tsx`

### Administration (3 files)

-   [ ] `src/Containers/Administration/Events/SearchFilterResourceType.tsx`
-   [ ] `src/Containers/Administration/Events/SearchFilterLevel.tsx`
-   [ ] `src/Containers/Administration/Events/SearchFilterDomain.tsx`

### Compliance Enhanced (1 file)

-   [ ] `src/Containers/ComplianceEnhanced/Coverage/components/CheckStatusDropdown.tsx`

### Policy Categories (1 file)

-   [ ] `src/Containers/PolicyCategories/PolicyCategoriesFilterSelect.tsx`

### Audit (1 file)

-   [ ] `src/Containers/Audit/ListeningEndpoints/ListeningEndpointsPage.tsx`

### Policies (7 files)

-   [ ] `src/Containers/Policies/Wizard/Step1/PolicyCategoriesSelectField.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step1/MitreTacticSelect.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step1/MitreTechniqueSelect.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step3/PolicyCriteriaFieldSubInput.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step3/PolicyCriteriaFieldInput.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step4/PolicyScopeForm.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step4/PolicyScopeCard.tsx`

### System Health (1 file)

-   [ ] `src/Containers/SystemHealth/DiagnosticBundle/DiagnosticBundleForm.tsx`

### Integrations (4 files)

-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/ExternalBackupIntegrations/S3CompatibleIntegrationForm.tsx`
-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/ApiTokenIntegrationForm/ApiTokenIntegrationForm.tsx`
-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/MachineAccessIntegrationForm.tsx`
-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/EmailIntegrationForm.tsx`

### Dashboard (2 files)

-   [ ] `src/Containers/Dashboard/NamespaceSelect.tsx`
-   [ ] `src/Containers/Dashboard/ClusterSelect.tsx`

### Main Page (1 file)

-   [ ] `src/Containers/MainPage/InviteUsers/InviteUsersForm.tsx`

### Clusters (3 files)

-   [ ] `src/Containers/Clusters/DiscoveredClusters/SearchFilterStatuses.tsx`
-   [ ] `src/Containers/Clusters/DiscoveredClusters/SearchFilterTypes.tsx`
-   [ ] `src/Containers/Clusters/InitBundles/InitBundleForm.tsx`

### Access Control (4 files)

-   [ ] `src/Containers/AccessControl/AuthProviders/RuleGroups.tsx`
-   [ ] `src/Containers/AccessControl/AuthProviders/AuthProviderForm.tsx`
-   [ ] `src/Containers/AccessControl/AuthProviders/ConfigurationFormFields.tsx`
-   [ ] `src/Containers/AccessControl/PermissionSets/PermissionsTable.tsx`

### Network Graph (9 files)

-   [ ] `src/Containers/NetworkGraph/components/EdgeStateSelect.tsx`
-   [ ] `src/Containers/NetworkGraph/components/DisplayOptionsSelect.tsx`
-   [ ] `src/Containers/NetworkGraph/components/DeploymentSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/common/AdvancedFlowsFilter/AdvancedFlowsFilter.tsx`
-   [ ] `src/Containers/NetworkGraph/common/NetworkPolicies.tsx`
-   [ ] `src/Containers/NetworkGraph/components/ClusterSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/components/TimeWindowSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/components/NamespaceSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/simulation/ViewActiveYAMLs.tsx`

### Components (7 files)

-   [x] `src/Components/PatternFly/FormMultiSelect.tsx`
-   [ ] `src/Components/PatternFly/CheckboxSelect.tsx`
-   [ ] `src/Components/PatternFly/RepeatScheduleDropdown.tsx`
-   [ ] `src/Components/PatternFly/DayPickerDropdown.tsx`
-   [ ] `src/Components/EmailNotifier/EmailNotifierForm.tsx`
-   [ ] `src/Components/SelectSingle/SelectSingle.tsx`
-   [ ] `src/Components/NotifierConfiguration/NotifierMailingLists.tsx`

## Migration Steps

1. **Update imports**: Change `from '@patternfly/react-core/deprecated'` to `from '@patternfly/react-core'`
2. **Update component usage**: The new Select components may have different props and behavior
3. **Test functionality**: Ensure all Select components work correctly after migration
4. **Update documentation**: Mark completed files with `[x]` in this checklist

## Notes

-   This analysis was generated automatically by scanning the codebase
-   All files listed contain imports from `@patternfly/react-core/deprecated`
-   Some files may require additional changes beyond just updating import paths
-   Consider testing each component individually to ensure compatibility

## References

-   [PatternFly Select Component Documentation](https://www.patternfly.org/components/forms/select)
-   [PatternFly Migration Guide](https://www.patternfly.org/get-started/migration-guide)

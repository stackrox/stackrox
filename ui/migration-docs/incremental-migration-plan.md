# Incremental Migration Plan - PatternFly Select Components

## Overview

This document outlines a phased approach to migrating all 56 files from deprecated `@patternfly/react-core/deprecated` Select imports to the modern `@patternfly/react-core` package. The plan is organized to minimize risk and dependencies.

---

## 📋 Migration Phases

### **Phase 1: Foundation Components (7 files)**

_Start here - these are reusable components that other files likely depend on_

**Target:** `src/Components/` directory

-   [ ] `src/Components/PatternFly/FormMultiSelect.tsx`
-   [ ] `src/Components/PatternFly/CheckboxSelect.tsx`
-   [ ] `src/Components/PatternFly/RepeatScheduleDropdown.tsx`
-   [ ] `src/Components/PatternFly/DayPickerDropdown.tsx`
-   [ ] `src/Components/SelectSingle/SelectSingle.tsx`
-   [ ] `src/Components/EmailNotifier/EmailNotifierForm.tsx`
-   [ ] `src/Components/NotifierConfiguration/NotifierMailingLists.tsx`

**Rationale:** Fix shared components first to avoid conflicts when updating dependent files.

---

### **Phase 2: Simple Feature Areas (8 files)**

_Low-risk, isolated features with minimal dependencies_

**Target:** Dashboard, System Config, and standalone features

-   [ ] `src/Containers/Dashboard/NamespaceSelect.tsx`
-   [ ] `src/Containers/Dashboard/ClusterSelect.tsx`
-   [ ] `src/Containers/SystemConfig/Form/FormSelect.tsx`
-   [ ] `src/Containers/SystemConfig/Form/SystemConfigForm.tsx`
-   [ ] `src/Containers/PolicyCategories/PolicyCategoriesFilterSelect.tsx`
-   [ ] `src/Containers/ComplianceEnhanced/Coverage/components/CheckStatusDropdown.tsx`
-   [ ] `src/Containers/MainPage/InviteUsers/InviteUsersForm.tsx`
-   [ ] `src/Containers/SystemHealth/DiagnosticBundle/DiagnosticBundleForm.tsx`

**Rationale:** These are likely standalone dropdowns with minimal complexity.

---

### **Phase 3: Administration & Events (4 files)**

_Administrative features - important but not user-facing_

**Target:** Administration directory

-   [ ] `src/Containers/Administration/Events/SearchFilterResourceType.tsx`
-   [ ] `src/Containers/Administration/Events/SearchFilterLevel.tsx`
-   [ ] `src/Containers/Administration/Events/SearchFilterDomain.tsx`
-   [ ] `src/Containers/Audit/ListeningEndpoints/ListeningEndpointsPage.tsx`

**Rationale:** Admin features are critical but used less frequently, good for mid-phase testing.

---

### **Phase 4: Clusters & Access Control (7 files)**

_Core infrastructure components_

**Target:** Cluster and access management

-   [ ] `src/Containers/Clusters/DiscoveredClusters/SearchFilterStatuses.tsx`
-   [ ] `src/Containers/Clusters/DiscoveredClusters/SearchFilterTypes.tsx`
-   [ ] `src/Containers/Clusters/InitBundles/InitBundleForm.tsx`
-   [ ] `src/Containers/AccessControl/AuthProviders/RuleGroups.tsx`
-   [ ] `src/Containers/AccessControl/AuthProviders/AuthProviderForm.tsx`
-   [ ] `src/Containers/AccessControl/AuthProviders/ConfigurationFormFields.tsx`
-   [ ] `src/Containers/AccessControl/PermissionSets/PermissionsTable.tsx`

**Rationale:** Important infrastructure, but contained within specific workflows.

---

### **Phase 5: Integrations (4 files)**

_External system integrations_

**Target:** Integration forms

-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/ExternalBackupIntegrations/S3CompatibleIntegrationForm.tsx`
-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/ApiTokenIntegrationForm/ApiTokenIntegrationForm.tsx`
-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/MachineAccessIntegrationForm.tsx`
-   [ ] `src/Containers/Integrations/IntegrationForm/Forms/EmailIntegrationForm.tsx`

**Rationale:** Isolated integration forms, lower risk of breaking core functionality.

---

### **Phase 6: Collections (4 files)**

_Data collection and rule management_

**Target:** Collections functionality

-   [ ] `src/Containers/Collections/CollectionResults.tsx`
-   [ ] `src/Containers/Collections/RuleSelector/ByNameSelector.tsx`
-   [ ] `src/Containers/Collections/RuleSelector/RuleSelector.tsx`
-   [ ] `src/Containers/Collections/RuleSelector/MatchTypeSelect.tsx`

**Rationale:** Grouped together as they're part of the same feature set.

---

### **Phase 7: Vulnerabilities (6 files)**

_Security vulnerability management_

**Target:** Vulnerability reporting and management

-   [ ] `src/Containers/Vulnerabilities/components/CVEStatusDropdown.tsx`
-   [ ] `src/Containers/Vulnerabilities/components/CVESeverityDropdown.tsx`
-   [ ] `src/Containers/Vulnerabilities/VulnerablityReporting/forms/CollectionSelection.tsx`
-   [ ] `src/Containers/Vulnerabilities/VulnerablityReporting/forms/ReportParametersForm.tsx`
-   [ ] `src/Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/ReportJobs.tsx`
-   [ ] `src/Containers/Violations/ViolationsTablePanel.tsx`

**Rationale:** Critical security features - handle with care and thorough testing.

---

### **Phase 8: Policies (7 files)**

_Policy wizard and management - high complexity_

**Target:** Policy creation and management

-   [ ] `src/Containers/Policies/Wizard/Step1/PolicyCategoriesSelectField.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step1/MitreTacticSelect.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step1/MitreTechniqueSelect.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step3/PolicyCriteriaFieldSubInput.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step3/PolicyCriteriaFieldInput.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step4/PolicyScopeForm.tsx`
-   [ ] `src/Containers/Policies/Wizard/Step4/PolicyScopeCard.tsx`

**Rationale:** Complex multi-step wizard - requires careful testing of the entire flow.

---

### **Phase 9: Network Graph (9 files)**

_Most complex - save for last_

**Target:** Network visualization and analysis

-   [ ] `src/Containers/NetworkGraph/components/EdgeStateSelect.tsx`
-   [ ] `src/Containers/NetworkGraph/components/DisplayOptionsSelect.tsx`
-   [ ] `src/Containers/NetworkGraph/components/DeploymentSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/components/ClusterSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/components/TimeWindowSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/components/NamespaceSelector.tsx`
-   [ ] `src/Containers/NetworkGraph/common/AdvancedFlowsFilter/AdvancedFlowsFilter.tsx`
-   [ ] `src/Containers/NetworkGraph/common/NetworkPolicies.tsx`
-   [ ] `src/Containers/NetworkGraph/simulation/ViewActiveYAMLs.tsx`

**Rationale:** Most complex feature with many interdependencies - handle last when you have experience from other phases.

---

## 🎯 **Recommended Workflow Per Phase**

1. **Update imports** in all files for the phase

    ```typescript
    // Change from:
    import { Select, SelectOption } from '@patternfly/react-core/deprecated';

    // To:
    import { Select, SelectOption } from '@patternfly/react-core';
    ```

2. **Run build** to catch any immediate TypeScript errors

    ```bash
    npm run build
    ```

3. **Test functionality** for that specific feature area

    - Manual testing of the UI components
    - Run relevant tests if available
    - Check for console errors

4. **Update checklist** in both markdown files

    - Mark completed files with `[x]`
    - Note any issues or special considerations

5. **Commit changes** for that phase

    ```bash
    git add .
    git commit -m "feat: migrate Phase X PatternFly Select components from deprecated imports"
    ```

6. **Move to next phase**

---

## 📈 **Benefits of This Approach**

-   **Progressive complexity** - Start easy, build confidence
-   **Isolated testing** - Test one feature area at a time
-   **Minimal conflicts** - Dependencies handled first
-   **Easy rollback** - Each phase can be committed separately
-   **Manageable scope** - Never more than 9 files at once
-   **Risk mitigation** - Critical features handled when you have most experience

---

## 🔄 **Phase Progress Tracking**

-   [ ] **Phase 1**: Foundation Components (7 files)
-   [ ] **Phase 2**: Simple Feature Areas (8 files)
-   [ ] **Phase 3**: Administration & Events (4 files)
-   [ ] **Phase 4**: Clusters & Access Control (7 files)
-   [ ] **Phase 5**: Integrations (4 files)
-   [ ] **Phase 6**: Collections (4 files)
-   [ ] **Phase 7**: Vulnerabilities (6 files)
-   [ ] **Phase 8**: Policies (7 files)
-   [ ] **Phase 9**: Network Graph (9 files)

**Total Progress**: 0/56 files completed (0%)

---

## 🚨 **Important Notes**

-   **API Changes**: The new Select components may have different props or behavior
-   **Testing Required**: Each phase should be thoroughly tested before proceeding
-   **Documentation**: Update any component documentation that references the old imports
-   **TypeScript**: Watch for type changes between deprecated and modern versions
-   **Backup**: Ensure you have a clean git state before starting each phase

---

## 📚 **Resources**

-   [PatternFly Select Migration Guide](https://www.patternfly.org/get-started/migration-guide)
-   [Modern Select Component Docs](https://www.patternfly.org/components/forms/select)
-   [PatternFly v5 Breaking Changes](https://www.patternfly.org/get-started/migration-guide#select)
